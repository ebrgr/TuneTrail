Faltava conferir se `DeleteListen` falha quando a escuta não existe. Isso define a gravidade do único ponto que encontrei. Conferido: `RemovePersonalListen` chama `DeleteListen` sem pré-checagem, então o caminho é raro. Segue o parecer.

## Parecer — ISSUE-139 (auditoria estática final)

**APROVADO.** Não há itens obrigatórios.

Rodei a consulta graphify exigida. Li o diff inteiro, `media_navigation.go` e os testes novos, e as linhas correspondentes de `internal/adminui/media.go` (102–180, 184–475, 478–715) e de `internal/adminui/catalog_review.go` (30–160). Não executei nada além da consulta.

### O que foi confirmado
- **Roteamento:** `MediaTaskAllowed` é a mesma lista em `IsPage`, `SafeReturn`, `media()` e na validação de `return_to` em `mediaAction`. Tarefas de escuta em álbum ou artista dão 404. Os POST continuam nos endpoints antigos; `/review` e `/confirm` não mudaram.
- **Endereço antigo:** a consulta é validada antes do redirecionamento, inclusive a busca da escuta em `listen_at` quando a tarefa é `""`. O redirecionamento mantém a consulta original. O índice não emite CSRF nem ticket, e as âncoras `aliases`, `metadata`, `artists`, `merge`, `artwork` e `delete` continuam existindo como `id` no índice.
- **Tickets:** só são criados em páginas com formulário de mutação (`merge` e `delete` só têm GET para `/review`). Os fragmentos de busca reaproveitam o ticket do cabeçalho, como na 124. A troca de ticket nos fragmentos POST continua igual.
- **Retorno:**
  - Se `return_to` vier preenchido mas inválido, a resposta é 400. Antes ele caía em silêncio para a página base.
  - `SafeHistoryReturn` limita o histórico ao perfil da rota.
  - `listen-delete` remove `listen_at` e troca `/listen` por `/listens`.
  - `historyMediaDestination` só redireciona para a gestão protegida quando a leitura pessoal falha, e mantém apenas `tz`.
- **Merge:** `confirmCatalog` usa `review.Plan.To`, que já vem invertido em `reviewCatalog:54-56`, então o sobrevivente fica correto também com `reverse=1`. Um sobrevivente que o perfil não ouviu vai para `/manage/<kind>/<id>`. A exclusão continua voltando a `/<slug>`.
- **Testes pelo router principal:** cobrem visitante e sessão inválida (com `next` exato), origem e `Sec-Fetch-Site`, CSRF ausente, errado ou duplicado, ticket ausente, errado, de outra sessão e reenviado, perfil por id e por slug, retorno externo, `//`, outro perfil e outra mídia. Também cobrem dois perfis com o mesmo instante, login voltando à tarefa, fragmentos e o caso do sobrevivente não ouvido.

### Apontamentos

**[SUGESTÃO] `internal/adminui/media.go:298-315`: em POST, a validação da consulta passou a valer para a consulta do `return_to`.**
A validação, com lista restrita de chaves, antes só rodava em GET; agora roda também em POST. Quando uma ação falha e o `return_to` é um histórico com filtros (`/second/listens?year=…&week=…&page=2&limit=1&track_id=N`, exatamente o caso de `validate_admin_webui.py`), as chaves `year`/`page`/… geram `ErrInvalidInput`. O resultado é um 400 em texto puro, quando o esperado seria a página de nova tentativa com a mensagem do erro real e um ticket novo. Isso não muda nenhum dado e é raro: `DeleteListen` não exige pré-checagem, e `confirm` é `required` no HTML. Para corrigir, basta aplicar a lista de chaves só quando a consulta vier de um caminho `/manage/...`, ou continuar usando `query` de `return_to` apenas para `tz`/`artist_query`/`merge_query`/`listen_at`/`history_return`.

**[SUGESTÃO] `internal/adminui/media_navigation.go:27-33`: `mediaDestination` trata qualquer erro como "sem leitura pessoal".**
Uma falha transitória do banco depois de um merge já aplicado manda o usuário para o índice protegido em vez de mostrar o erro. É seguro, porque o destino exige sessão, e o comentário já diz isso. Se quiser distinguir os casos, use `errors.Is(err, db.ErrNotFound)` para o fallback.

**[SUGESTÃO] `internal/adminui/media.go:193-213`: a tarefa é extraída do caminho de duas formas diferentes.**
Em GET ela sai de `r.URL.Path`; em POST, do caminho de `return_to`. Os dois usam `len(parts) == 5`, sem conferir o prefixo `expected`. Hoje não há falha, porque `mediaAction` já restringe o `return_to` a `expected`, `expected/<tarefa permitida>` ou ao histórico do perfil, que tem 1 a 3 segmentos. Um helper `taskFromPath(expected, path)` deixaria essa invariante explícita e evitaria que ela quebre se o histórico ganhar caminhos de 5 segmentos.

**[SUGESTÃO] `assets/templates/admin/manage.html`, no índice: âncoras vazias.**
`<span id="metadata"></span><span id="artists"></span>` existem só para preservar as âncoras antigas. Um comentário de template dizendo isso evitaria que alguém os remova numa limpeza.

### Escopo e artefatos
- Não há diálogos, ícones, drop nem fragmento inicial novo. A arte da faixa continua indo para o álbum (`album ID 11`, testado). O catálogo é global e as escutas são pessoais (`/default/.../edit` responde 200 e `/default/.../listen` responde 404).
- Nas referências públicas, a única mudança é `data-ui-generation`, coerente com a regeneração pelo comando oficial. As referências administrativas refletem o índice novo, e o "Cancel" do merge aponta para `/merge`.
- O CHANGELOG e as chaves i18n (en/pt-BR) estão de acordo com a mudança.
