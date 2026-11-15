# ISSUE-139 — Páginas de gestão por tarefa

Implementação em `feat/issue-139-paginas-de-gestao-por-tarefa`, iniciada por `make issue.start ISSUE=139`. Escopo: recorte A da ISSUE-108 e decisões registradas nas duas tarefas, sem alterar as ISSUE-108/140–142.

## Mudanças

A base `/{slug}/manage/{tipo}/{id}` é um índice curto. As páginas `/edit`, `/image`, `/merge` e `/delete` atendem artista, álbum e faixa; `/add-listen`, `/listens` e `/listen?listen_at=...` atendem somente faixa. A lista/exclusão de escutas fica separada da adição; a escuta selecionada tem só sua confirmação individual. A imagem da faixa continua usando o pipeline de capa do álbum.

Os blocos de `assets/templates/admin/manage.html` são reaproveitados. Não há CSS, JavaScript, diálogo, ícones, gesto de arrastar ou novo protocolo de fragmento nesta entrega. As traduções novas estão nos catálogos inglês e português.

As consultas antigas encaminham para a tarefa: `listen_at` → `/listen`, `artist_query` → `/edit`, `merge_query` → `/merge`, e `tz` isolado de faixa → `/listens`. Na combinação de consultas, essa é a prioridade; todos os parâmetros validados são preservados. Âncoras antigas apontam para links do índice: fragmentos não chegam ao servidor e não exigem JavaScript para acessar a tarefa. `/review` e todos os endpoints POST mantêm seus contratos. O review continua usando seu plano próprio, vinculado a sessão/perfil/rota e revalidado na transação.

Rotas e retornos compartilham a allowlist de tarefas por tipo. Login retorna à URL da tarefa e seu contexto. POST normal volta à tarefa correspondente, preservando consulta/âncora, erro e valores; excluir uma escuta selecionada sem retorno de histórico volta à lista, retirando `listen_at`. O retorno pessoal aceita home, histórico e páginas de mídia do mesmo perfil, corrigindo também o link antigo de histórico embutido em mídia; caminhos externos, protocol-relative e outros perfis são rejeitados.

Merge confirmado vai ao sobrevivente efetivo do plano, inclusive na direção invertida. Decisão confirmada pelo coordenador em 15/11/2026 (resposta `msg_70e77cc2cb9a`): usa a página pública quando há relação de escutas no perfil; caso contrário, usa o índice administrativo do sobrevivente, sem ampliar a leitura pública. Exclusão de catálogo sem sobrevivente vai ao perfil. Sessão, origem, CSRF, ticket de submissão e escopo pessoal continuam aplicados no servidor. Índice, redirecionamentos e páginas de busca/revisão de catálogo não desperdiçam tickets de mutação; buscas parciais reutilizam o ticket existente e POST parcial renova o consumido, conforme a ISSUE-124.

## Verificação

- `make check`: aprovado, incluindo vet, testes Go, testes de interface/estilo, referência HTML e **12/12 jornadas E2E** em dados descartáveis e portas efêmeras. O gate é repetido pelo comando obrigatório `make issue.review` sobre o diff final.
- `engine/adminui_tasks_139_test.go`: testa pelo roteador principal real, que instancia `RequireSubmissionToken: true`; separação por tarefa/tipo, índice e âncoras, consultas antigas, GET/HEAD, login completo com retorno, visitante e sessão inválida, origem estrangeira/Fetch Metadata, CSRF ausente/incorreto/duplicado, ticket ausente/incorreto/reutilizado/de outra sessão, IDs/slugs forjados, retorno externo/protocol-relative/de outro item ou perfil e erro com valores/ticket novo.
- Os testes também cobrem catálogo global sem escutas no perfil, dois perfis com a mesma faixa e instante, exclusão pessoal isolada, retorno a mídia, fragmentos de busca/POST com marcadores e rotação da ISSUE-124, merge normal/invertido e exclusão sem sobrevivente.
- Testes anteriores de gestão/CSRF/fragmentos foram atualizados para a navegação por tarefa; a jornada E2E administrativa e a migração verificam os redirecionamentos antigos e o sobrevivente do merge.
- Referência regenerada em diretório novo com `make ui.reference OUT=/tmp/tunetrail-issue139-reference-final` e copiada para `backlog/evidencias/issue-067/reference`. As três páginas administrativas agora mostram índices; o cancelamento do review volta à tarefa. Nas páginas públicas, mudou somente `data-ui-generation`, pois o template integra o snapshot de assets; nenhum CSS público foi alterado.
- Revisão deslop: conferida a composição de fragmentos legados e removida a busca duplicada de artistas após POST; regras de tarefa/retorno ficam compartilhadas. `git diff --check` sem problemas.
- Graphify atualizado pelo script AST oficial, sem extração semântica nem API. O aviso do extrator sobre `tokens.css` classificado como sensível é preexistente à mudança; o rebuild terminou com sucesso.

## Critérios e limites

Critérios #1–#4 verificados pelos testes acima. Aceite funcional/visual final pertence ao usuário; não há alteração de status para Done nem publicação, push ou merge. Os detalhes ampliados e diálogo da escuta pertencem à ISSUE-141; aqui permanece a identificação por faixa/instante e a confirmação individual existente. A atualização de capa sem recarga no diálogo pertence à ISSUE-140; esta entrega preserva o POST de imagem e seu retorno de página completa.

Uma execução intermediária do gate encontrou um arquivo de teste temporário criado/removido pelo Reviewer durante a varredura do Go; a revisão foi interrompida, reiniciada com permissões somente de leitura e o gate completo foi repetido sequencialmente sem esse arquivo.

O gate inicial apontou testes/E2E que ainda procuravam formulários na base e a referência anterior, além de três linhas HTML acima do limite; as expectativas foram atualizadas para o comportamento intencional. Uma primeira execução manual de `go test` sem as variáveis de assets/catálogos falhou por configuração da invocação; o gate oficial exporta as variáveis e passou.

## Reviewer

Parecer do Claude Code CLI, modelo Opus com esforço medium, solicitado conforme `.agents/reviewer.md`, em `reviewer.md`. A primeira revisão apontou dois problemas: destino público inacessível para sobrevivente sem escutas no perfil e `listen_at` obsoleto no retorno de um formulário antigo de exclusão. Ambos foram corrigidos e cobertos pelo roteador principal, incluindo artista/álbum/faixa com e sem escutas no perfil. Também foram separados os grupos de imports e a busca da escuta selecionada ficou restrita à tarefa de escuta/encaminhamento legado, conservando a validação do parâmetro nas outras tarefas.

A mesma política de sobrevivente é aplicada ao retorno a uma mídia após excluir sua última escuta pessoal: se a leitura pública deixar de existir para o perfil, o retorno usa a gestão protegida e conserva o fuso. Filtros do histórico público são descartados nesse fallback porque não se aplicam ao índice administrativo.

O retorno legado de `listen-delete` retira `listen_at` em qualquer tarefa administrativa do mesmo item, incluindo base convertida em `/listens`; assim o destino não tenta carregar a escuta recém-excluída. Revisão final após as correções em `reviewer-final.md`: **APROVADO, sem itens obrigatórios**. A primeira sugestão dessa revisão também foi corrigida: erros de POST com retorno ao histórico público filtrado agora mantêm o formulário, a mensagem, o ticket renovado e o retorno original; filtros públicos não são tratados como parâmetros da gestão. Teste `TestManagementListenErrorReturn139` pelo roteador principal; auditoria desse delta em `reviewer-complementar.md`: **APROVADO, sem itens obrigatórios**. O teste foi executado e aprovado pelo implementador.
