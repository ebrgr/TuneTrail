# Avaliação da ISSUE-108 — Codex

Data: 15/11/2026. Base inspecionada: `3d9ce50` (`ISSUE-068: Publica versão 0.20`), árvore inicialmente limpa. Avaliação estática, sem implementação, execução de testes, servidor ou novas capturas. Foram lidos a tarefa pelo Backlog.md, AGENTS.md, CLAUDE.md, a comparação Koito × TuneTrail e scripts/README.md. Navegação inicial pelo graphify, seguida das fontes localizadas e dos caminhos referenciados por elas. Foi consultada também a orientação local `ui-craft` para avaliar diálogo, foco e estados.

**Recomendação: dividir, após ajustar a descrição.** O objetivo permanece válido; a infraestrutura das ISSUE-109/124 já resolve parte importante do trabalho. A entrega restante reúne separação de rotas, integração de formulários em documento público, ciclo de vida de diálogos e tickets, atualização da página de leitura e upload com prévia. São mudanças relacionadas, mas com contratos e verificações distintos. Sugiro quatro recortes sequenciais, detalhados abaixo; não criar novas tarefas nem modificar a 108 nesta avaliação.

## 1. Estado atual conferido no código

### Cabeçalho, gestão e histórico

- O link público **Manage this item** continua incondicional em `assets/templates/public/media.html:40`. O MusicBrainz já existe para todos quando há MBID (`:41–43`); URL construída por tipo em `internal/webui/media.go:94–96`. Falta transformar esses links em barra e condicionar os de gestão a `.Admin`.
- A gestão continua em `/{slug}/manage/{kind}/{id}`. O roteador de páginas reconhece a base e `/review` (`internal/adminui/routing.go:22`), e o dispatcher envia a gestão para `media` (`internal/adminui/admin.go:205–208`). O handler limita o caminho a quatro/cinco segmentos e aceita GET da base, GET `/review`, POST `/confirm` e ações POST (`internal/adminui/media.go:99–175`). **Não existem as páginas GET `/edit`, `/add-listen`, `/image`, `/merge`, `/delete`.**
- O template monolítico já está separado em blocos reaproveitáveis: aliases, metadata, artists e listens (`assets/templates/admin/manage.html:45–49`); escuta selecionada em `:50–72`; merge em `:73`; artwork em `:74–112`. As definições começam em `:131`, `:185`, `:210`, `:278` e `:333`. Ainda não são páginas por tarefa: `manage/listens` mistura lista/exclusão e adição; `manage/merge` mistura exclusão de catálogo e mesclagem (`:278–373`). A quantidade “42 formulários” descreve a fixture histórica, não um número fixo da aplicação: há formulários gerados por aliases, artistas e escutas.
- O link **Manage** de cada escuta também permanece público, usando `listen_at`, `history_return` e `#selected-listen` (`assets/templates/public/live.html:332–340`). A consulta da escuta selecionada já restringe perfil, faixa e instante (`internal/adminui/media.go:330–345`). Seu formulário separado dentro da página grande possui confirmação e tokens (`assets/templates/admin/manage.html:54–69`).
- Há um detalhe para corrigir ao redesenhar o histórico: esse mesmo partial aparece na página de mídia (`assets/templates/public/media.html:69`) e envia `CurrentURL` como retorno; o handler só permite `history_return` para a home ou `/listens` do mesmo perfil (`internal/adminui/media.go:351–359`). Portanto um retorno vindo da própria página de mídia pode ser rejeitado. É uma inferência diretamente do código, não uma reprodução em navegador. O novo contrato deve contemplar explicitamente as origens suportadas.

### `.Admin`, cache e live.js

- A ISSUE-109 já está integrada: `LivePage.Admin` é documentado em `internal/webui/live_models.go:40–41`, preenchido antes do despacho das páginas (`internal/webui/handlers.go:69–70`) e obtido exclusivamente de sessão válida de administrador (`:228–238`). A API key de ingestão não substitui essa sessão.
- O `<body>` recebe `data-admin` somente para administrador (`assets/templates/public/live.html:19`). Respostas públicas 200 já têm `Cache-Control: private, no-cache`, `Vary: Cookie` e `X-TuneTrail-Admin` (`internal/webui/handlers.go:267–280`). Páginas administrativas e fragmentos usam `no-store` (`internal/adminui/admin.go:155`; `internal/adminui/media.go:86–88`). Não relaxar fragmentos com formulários para o cache dos documentos públicos.
- `live.js` **já deixou de consultar `/apis/web/v1/user`**. Usa `data-admin` e adiciona controles imediatamente na ativação inicial (`assets/js/live.js:69–81`). Ao restaurar bfcache, faz HEAD e compara sessão, geração e preferências (`:83–105`), removendo controles em `pagehide` (`:109–114`). Essa requisição de revalidação é intencional; “sem requisição extra” deve significar sem consulta tardia de papel na primeira carga.
- A exclusão rápida existente ainda usa DELETE na API de escutas do perfil e recarrega a página (`assets/js/live.js:116–134`), não o formulário HTML com CSRF/ticket. É um caminho distinto; reaproveitar `.Admin` não o converte no diálogo proposto.

### Fragmentos, tickets e scripts

- O contrato não é simplesmente acrescentar `?partial=1`: exige `X-TuneTrail-Partial: 1` e recusa identificar navegação/documento como fetch (`internal/adminweb/fragment.go:5–10`). Hoje o GET parcial da gestão cobre apenas buscas `artist_query` (faixa) e `merge_query`; navegação sem header redireciona à página completa (`internal/adminui/media.go:156–175`).
- POST aprimorado devolve somente aliases, metadata, artists ou listens; artwork fica de fora por decisão anterior do usuário (`internal/adminui/media.go:53–66`, `:563–582`). A resposta informa `X-TuneTrail-Fragment` e ticket novo (`:77–88`). Há preservação de erro/valores e status (`:398–418`).
- O CSRF é reutilizável até expirar e ligado à sessão (`internal/adminweb/csrf.go:69–86`). O ticket de submissão é **outro token**, aleatório, consumido uma vez, vinculado à sessão, com TTL de 30 minutos e limite global de 4096 (`internal/adminui/submission.go:14–51`). Não é vinculado a item, tarefa ou perfil; essa validação é responsabilidade da rota/formulário. Em produção ele é obrigatório (`engine/routes.go:183`).
- Um GET de busca parcial não emite ticket: só reaproveita UUID recebido em `X-TuneTrail-Submission`; a existência/validade real será conferida no POST (`internal/adminui/media.go:188–198`). Isso serve para buscas dentro de um editor já aberto. **Abrir o primeiro diálogo numa página pública sem formulários precisa de um contrato novo que emita ticket**, não pode copiar cegamente esse fluxo de busca.
- `enhance.js` é o mecanismo efetivo da ISSUE-124: serializa pedidos com um ticket compartilhado, troca seção pelo ID, renova tickets em todos os formulários e retorna foco (`assets/js/enhance.js:12–23`, `:25–70`). Mas depende de formulários presentes na inicialização (`:5–6`), captura base/status uma única vez (`:5`, `:10`), usa `history.replaceState` na busca (`:57–62`) e atualiza o primeiro `h1` (`:54`). Copiado para a página pública, pode deixar de inicializar, mudar a URL da mídia para a gestão ou sobrescrever o título errado. Precisará de escopo explícito ao editor/diálogo e atualização separada do cabeçalho público.
- `ui.js` já abre `<dialog>` com `showModal()` e devolve foco, mas registra os listeners de fechamento apenas nos diálogos presentes na inicialização (`assets/js/ui.js:5–31`). É uma base útil, não um carregador remoto de tarefas. `search.js` demonstra cancelamento e descarte de resposta antiga (`assets/js/search.js:25–45`, `:79–87`); seu combobox é específico da busca global.
- `admin.js:15` aceita um arquivo arrastado **sobre o formulário existente**, preenche input e dispara `change`. Não faz prévia, confirmação, upload automático nem escuta a capa pública; o registro também ocorre só no carregamento. Além disso `admin.js`, `enhance.js` e `admin.css` só são carregados no head administrativo; páginas públicas recebem `ui.js`, `live.js` e `search.js` (`assets/templates/shared/head.html:19–28`). Importar `admin.js` inteiro traria o comportamento global de ocultar body/recarregar bfcache (`assets/js/admin.js:8–13`). Extrair/reusar a função necessária é preferível.

## 2. Passos 1 a 7: validade e trabalho restante

| Passo | Já existe | Falta / risco / atualização da descrição |
| --- | --- | --- |
| 1 — barra | Capa, cabeçalho, MBID e ExternalLink; inventário no manifest (`assets/manifest.json:15–34`). | Barra ordenada, Pencil/Image/Merge/Plus/Trash2, traduções, tokens de 30/22px e alvos 44px. Registrar novos ícones no manifest, não apenas copiá-los. MusicBrainz é link público, não ação administrativa. A posição desejada continua válida; coordenar com ISSUE-073 para não ampliar esta tarefa à reconstrução do cabeçalho. |
| 2 — permissão | `.Admin`, cache e consumo pelo live.js já prontos, conforme seção 1. | Usar o sinal nos links de mídia/histórico e nos novos gatilhos. Remover do escopo a implementação do sinal e a remoção do fetch tardio, já feitas. Testar ausência no HTML de visitantes, não só ocultação por CSS. |
| 3 — páginas | Blocos de edição, serviços, review e confirm existentes. | Novas páginas, modelos mínimos, índice e compatibilidade de query/âncoras antigas. Separar adição de escuta da lista e exclusão da busca de merge. `IsPage`, despacho, allowlist de query, `SafeReturn` e validação de `return_to` precisam evoluir juntos. Não basta adicionar cinco templates. Decidir destino da imagem de faixa, hoje edição da capa do álbum. |
| 4 — diálogo | `<dialog>` e foco já usados; protocolo parcial disponível. | Carregador por tarefa, fragmento inicial com tokens, título acessível, loading/erro, fallback navegável e cancelamento ao fechar/trocar. A ISSUE-107 não oferece diálogo remoto pronto; citar também 124. Links devem conservar href real, inclusive cliques modificados/abrir em nova aba. Evitar diálogos aninhados durante revisão. |
| 5 — atualizações | POST por seção e erros locais da 124. | Adaptar ao documento público e refletir aliases/nome, artistas, MBID, contagens/escutas e capa conforme a ação; definir quais áreas são atualizadas sem perder rolagem. Hoje merge/delete confirmados vão sempre à home do perfil (`internal/adminui/catalog_review.go:156–157`), não ao sobrevivente/lista. O texto “como hoje” precisa especificar esse destino. Artwork foi deliberadamente excluído da atualização parcial; qualquer mudança desse acordo precisa de decisão. |
| 6 — arrastar capa | Upload seguro e drop no formulário. | Drop na capa somente para admin e álbum/artista, prévia, confirmar/cancelar, acessibilidade alternativa, validação e tratamento de falha. Não há integração pronta. Se a confirmação acontecer no mesmo diálogo da barra, evitar pipeline de upload paralelo. |
| 7 — escuta | Identificação por perfil/faixa/instante, confirmação de exclusão e retorno. | Página/fragmento exclusivos de escuta, abertura no histórico e atualização/remoção de linha e estados vazios. Definir “ver detalhes” (campos ainda não especificados), visibilidade ao visitante e convivência com o × do live.js. Incluir histórico dentro das páginas de mídia e suas URLs de retorno. |

## 3. Critérios #1 a #7

| Critério | Avaliação e verificação necessária |
| --- | --- |
| #1 | Pendente: barra e layout não existem. Verificar três tipos com/sem MBID, admin/visitante, desktop/celular, ordem do DOM coerente com visual, toque/foco e texto longo traduzido. Usar tokens e camadas atuais. |
| #2 | Parcial: infraestrutura e primeiro consumo no live.js prontos; links continuam públicos. Verificar HTML e primeira pintura sem consulta de papel; preservar HEAD de bfcache e rejeição real nas rotas. |
| #3 | Pendente: há blocos aprimorados, sem páginas por tarefa nem diálogo remoto. MusicBrainz é exceção externa explícita; ícones de gestão abrem só seu fluxo, com página equivalente sem JS. Verificar erros, sessão expirada, fechar durante fetch, reabrir e foco. |
| #4 | Pendente: base continua editor completo. “Endereços antigos” deve incluir `listen_at`, `history_return`, `artist_query`, `merge_query`, `tz`, `/review` e ações POST, além das âncoras. Uma base transformada em índice sem encaminhar consultas antigas perderia funcionalidades. |
| #5 | Pendente: drop no formulário não satisfaz capa + prévia + confirmação. Verificar cancelamento sem POST, arquivo inválido/grande, falha de rede, repetição, sessão expirada e uso pelo teclado via página de imagem. |
| #6 | Pendente para novas rotas: já existem testes de base reutilizáveis; ampliar matriz abaixo com tickets obrigatórios. Teste só do Handler não prova que o roteador principal reconhece o novo caminho. |
| #7 | Pendente: comparação é evidência histórica da v0.8. Fazer capturas atuais lado a lado com a referência Koito identificada, três mídias, barra e diálogos em desktop/celular; registrar estados e diferenças intencionais. Aceite visual depende do usuário. |

Os critérios atuais não tornam verificáveis várias promessas dos passos 5 e 7. Acrescentar explicitamente atualização coerente da página pública e fluxo de escuta por tarefa; incluir também recuperação de erro/rede, foco e compatibilidade de URL. Não marcar critérios como atendidos só pela infraestrutura existente.

## 4. Segurança: preservar os contratos, testar as novas superfícies

1. **Sessão e autorização:** revalidar a cada GET/HEAD/POST e fragmento via `SessionIdentity`, não confiar em `.Admin`/DOM. Cookies ambíguos são rejeitados (`internal/adminweb/cookies.go:30–40`), papel validado no serviço (`internal/admin/auth.go:49–61`). Leitura protegida de visitante pode continuar redirecionando a login; não deve entregar formulário/tokens no corpo. Testar visitante, cookie inválido/duplicado, sessão expirada/revogada, não admin, chave de ingestão e troca de sessão entre abrir e enviar.
2. **Origem e CSRF:** manter o middleware administrativo (`internal/adminui/admin.go:155`). A política atual verifica Origin e Fetch Metadata em mutações, não em GET/HEAD (`internal/adminweb/origin.go:15–43`); não descrever `partial` como autorização ou proteção de origem. `ParseSessionForm` usa corpo, recusa CSRF por query/header e limita envelope (`internal/adminweb/forms.go:39–45`, `:70–86`). Testar origem estrangeira, `Sec-Fetch-Site: cross-site`, tokens ausentes/duplicados/errados/de outra sessão, GET sem mutação e métodos inválidos. Upload deve conservar CSRF válido antes de spool e os limites de admissão/tamanho (`internal/adminui/media.go:424–440`).
3. **Ticket de submissão:** não confundir com CSRF nem com plano de merge/delete. Conservar consumo antes da mutação, ticket novo após sucesso ou erro de validação e nenhuma repetição automática de POST quando o resultado é incerto (`assets/js/enhance.js:47–52`). Testar replay, duas ações concorrentes, sessão diferente, expiração, múltiplas abas/diálogos e fechar/reabrir durante resposta. Não emitir tickets a cada tecla de busca; limite global torna isso esgotável. Formulários novos devem sempre carregar ticket em produção; o GET inicial do diálogo deve emitir o primeiro.
4. **Plano destrutivo:** review possui token próprio ligado a sessão, perfil, rota e tipo, TTL de dez minutos/limite de 256 (`internal/adminui/catalog_review.go:87–101`); confirmação consome e revalida plano na transação (`:115–153`). Não trocar esse plano por confirmação só no cliente nem pelo ticket genérico. Testar plano antigo após mudança no catálogo, replay, inversão de merge, troca de destino/tipo/rota/perfil e fechamento/reabertura. GET review hoje rejeita parâmetros extras (`:36–40`): `partial=1` exige tratamento deliberado, não adição transparente.
5. **Perfil versus catálogo global:** as ações conferem `profile_id` + `profile_slug` contra o perfil da rota (`internal/adminui/media.go:455–457`). Catálogo é compartilhado (`:199` usa `WithoutProfile`; aviso em `assets/templates/admin/manage.html:35–43`), enquanto adicionar/excluir escuta é pessoal (`internal/admin/listens.go:65–99`). Não impor incorretamente que mídia só possa ser editada se ouvida nesse perfil; impedir, sim, que uma escuta de outro perfil seja exposta/alterada. Testar dois perfis com mesma faixa/instante e IDs/slugs forjados, retornos cruzados e mídia não ouvida.
6. **Fragmentos e retorno:** usar `no-store` para qualquer conteúdo com formulário/token; escapar por html/template; validar marcador e estrutura antes do swap, sem inserir login/documento completo dentro do diálogo. GET com `partial=1` digitado deve fornecer a página canônica completa. Alterar juntos `internal/adminui/routing.go:22`, `engine/routes.go:198–212`, `internal/adminweb/return.go:48–60` e `internal/adminui/media.go:445–452`: hoje `SafeReturn` e o alvo esperado não aceitam as novas páginas por tarefa. Testar redirects externos/protocol-relative, caminhos codificados, queries duplicadas/desconhecidas, HEAD e cache de admin/visitante.
7. **Ciclo de vida da página pública:** os formulários com segredos vão passar a existir em documento elegível a bfcache. O live.js hoje só remove controles da tabela e revalida no retorno; o admin.js usa outra estratégia. Fechar/limpar diálogos sensíveis, abortar pedidos e revalidar antes de reativar. Testar logout/expiração em outra aba, voltar/avançar, rede indisponível e resposta antiga após fechar/trocar de item. Não permitir resposta de contexto anterior preencher diálogo novo.
8. **Imagem:** reutilizar `ReplaceArtwork` e os validadores do servidor, não criar endpoint de upload independente. O serviço autoriza, limita bytes e resolve a imagem de faixa para o álbum (`internal/admin/artwork.go:77–114`). Preview local não valida arquivo; testar tipos aceitos/rejeitados, múltiplos arquivos, limite, cancelamento sem envio, URL versus arquivo simultâneos e falha parcial. Liberar object URLs ao trocar/cancelar prévia.

Testes já localizados, mas não executados nesta avaliação: `engine/admin_badge_109_test.go:14–84` (sessão/cache/HEAD), `engine/adminui_sections_124_test.go:64–145` (âncoras, ticket renovado/replay, erro, buscas/navegação), `engine/adminui_media_test.go:45–98` (CSRF, perfil, DST e isolamento) e `engine/adminui_catalog_review_test.go:52–101` (plano, sessão/rota, stale, replay). O teste da 124 liga `RequireSubmissionToken: true` (`:28`); vários testes antigos usam o default opcional. Novos testes devem usar configuração equivalente à produção e passar pelo router nas verificações de rota.

## 5. Recortes sugeridos, nesta ordem

### A. Páginas de gestão por tarefa e compatibilidade dos links antigos

**Escopo:** separar modelos/blocos e GETs edit/add-listen/image/merge/delete e uma página de escuta específica; transformar base em índice; preservar endpoints de mutação/review e semântica de perfil. Preparar blocos para fragmentos, mas concluir primeiro navegação normal sem JS.

**Critérios verificáveis:** cada página contém apenas sua tarefa; tarefas inválidas para o tipo não são ofertadas/aceitas; base oferece índice; URLs antigas com consulta levam à tarefa correspondente sem perder fuso/escuta/retorno; POST correto volta à tarefa, preserva erros/valores; login retorna à nova URL; catálogo global versus escuta pessoal explícito; matriz de sessão/origem/CSRF/tickets/perfil passa pelo router.

**Dependências:** 124, 107, 106, 109 concluídas e estrutura de assets/CSS da 068. Nenhuma dependência nova de visual.

### B. Barra de ações e diálogos de tarefa sobre as páginas de mídia

**Escopo:** ícones/traduções/tokens, condição `.Admin`, carregador de fragmento inicial, sessão/tickets, adaptação do enhance.js ao contêiner, revisão destrutiva dentro do fluxo e atualização das informações visíveis após edição. Mesma página de A fornece conteúdo completo ou fragmento; MusicBrainz conserva link externo.

**Critérios verificáveis:** ordem/posição em três tipos e duas larguras; visitante recebe só ações públicas; href funciona sem JS; abrir/fechar/Tab/Esc/restaurar foco; loading/erro/rede/sessão expirada recuperáveis; não muda URL pública durante busca nem sobrescreve h1 indevido; nova submissão usa ticket renovado; título/MBID/artistas e dados afetados ficam coerentes; merge/delete navegam ao destino aprovado; bfcache não restaura formulário ativo antigo; capturas e aceite visual dessa superfície.

**Dependências:** A. Alinhar geometria do cabeçalho com ISSUE-073, sem incorporar degradê/cor/capa nova nesta entrega.

### C. Escuta individual no histórico, com página e diálogo equivalentes

**Escopo:** usar página de escuta de A e carregador de B no link Manage, definir detalhes, excluir apenas o registro selecionado, reconciliar × existente do live.js e retornos na home/listens/mídia.

**Critérios verificáveis:** mesmo perfil/faixa/instante na página e no fragmento; exclusão com confirmação não altera outro perfil; linha/contagens/estado vazio e eventual paginação coerentes; rolagem/foco preservados; sem JS retorna ao contexto aprovado; visitante não recebe controle de gestão; testes de retorno vindo de mídia e de dois perfis; captura do diálogo.

**Dependências:** A e B. Separá-lo evita esconder o passo 7 atrás do aceite da barra.

### D. Troca de capa por arrastar, com prévia e confirmação

**Escopo:** ampliar a tarefa imagem de A/B para aceitar drop sobre capa de álbum/artista, usar o mesmo formulário/pipeline seguro e oferecer prévia/confirmar/cancelar.

**Critérios verificáveis:** nenhum envio antes de confirmar; cancelar preserva capa; teclado/input mantêm alternativa; arquivo inválido, grande, múltiplo, sessão expirada e falha de rede têm resposta clara; sem retry automático; ao concluir, imagem e srcset corretos e política de recarga aprovada; foco e object URLs limpos; desktop/celular com capturas e aceite.

**Dependências:** A e B; C não é requisito técnico. O fluxo de imagem já deve funcionar por página/ícone em B; D acrescenta o gesto.

Essa divisão preserva entregas úteis e revisáveis, em vez de uma primeira entrega apenas estrutural sem uso. Se o usuário preferir uma única issue, adotar esses quatro marcos internos com aceite funcional explícito; o risco de regressão continua maior porque interliga página pública e administração.

## 6. CSS, assets e evidência visual na v0.20

O caminho vigente é `assets/`, não os caminhos antigos `internal/*/templates` da comparação histórica. O manifest mapeia os templates (`assets/manifest.json:5`, `:76–77`) e ordena partes CSS (`:52–70`). Barra/capa responsiva pertencem a `assets/css/ui/pages/media.css`, já em `@layer pages` (`:1–29`); primitivas compartilhadas de diálogo em `assets/css/ui/components/dialog.css`, `@layer components` (`:1–24`); tokens em `assets/css/ui/tokens.css`; regras administrativas específicas ficam sob `assets/css/admin/`. Não acrescentar tudo a um ui.css monolítico nem competir por especificidade entre camadas sem considerar a cascata. Limite e política estão em `scripts/README.md:72`.

A atualização de referências deve refletir mudança intencional, não tentar manter igualdade de HTML: `scripts/README.md:78` descreve o gate atual. A comparação computada da 068 não cobre foco/hover/diálogo aberto (`:70`), portanto não substitui verificar esses estados. Referência Koito atual para capturas precisa ser identificada; este relatório usa a comparação entregue como contexto histórico, sem consultar servidor operacional ou porta 4110.

## 7. Decisões que só o usuário pode tomar

1. Aprova dividir nos recortes A–D, ou prefere uma issue única com esses marcos? É decisão de planejamento, não limitação técnica.
2. A imagem continua com recarga completa, decisão registrada no código da 124, ou o novo diálogo deve permanecer aberto e atualizar a capa sem recarga? O passo 5 hoje conflita com essa exceção.
3. A faixa perde o acesso à imagem que hoje altera o álbum, ou a tarefa continua disponível no índice/página mesmo sem ícone de imagem? Não remover silenciosamente essa capacidade.
4. Após merge/delete, manter home do perfil (comportamento atual) ou ir ao sobrevivente/lista específica? Qual lista e qual preservação de filtros?
5. Quais “detalhes” da escuta devem aparecer? O link Manage some para visitantes ou eles ganham uma ação pública de detalhes separada? O × rápido continua coexistindo ou é substituído pelo novo fluxo?
6. Compatibilidade antiga inclui consultas, âncoras e URLs de ação/review, conforme proposto em A? Recomendo preservar esses contratos, não somente fazer a base retornar 200.
7. Qual versão/instância e quais imagens do Koito são a autoridade para o aceite visual, e onde a ISSUE-073 poderá alterar a posição da barra? A comparação de 13/11 define ordem/tamanhos, mas não é captura atual desta implementação.

Detalhes de implementação como renovar tickets, não confiar no DOM, manter allowlists e separar escuta de catálogo não exigem decisão do usuário: são requisitos de correção. A revisão Advisor indicada na tarefa deve ocorrer na implementação conforme as regras locais; esta avaliação foi individual, conforme o despacho, sem trabalhadores auxiliares.

## 8. Limites e ocorrências da análise

Nenhum código/template/CSS/teste/tarefa foi alterado; sem commit, push, merge, mudança de status ou servidor. Não foi reconstruído o graphify porque não houve modificação de código. A única escrita autorizada é este relatório.

A consulta inicial ao graphify e a consulta ampliada tiveram saída truncada por orçamento (aviso do próprio CLI); a análise continuou por símbolos/caminhos específicos e pelo JSON do grafo, sem tratar a saída truncada como inventário completo. A leitura opcional de `.../graphify-out/wiki/index.md` falhou porque o arquivo não existe; foram usadas as consultas em vez de wiki. `printenv ORCA_CLI_COMMAND ORCA_DEV_REPO_ROOT` retornou 1 sem conteúdo, indicando variáveis não definidas; o CLI resolvido foi `orca`, cujo guia de orquestração carregou normalmente. A primeira leitura conjunta de skills/documentos também teve saída truncada; trechos relevantes foram consultados separadamente. Essas ocorrências não impedem as conclusões; não houve medição de renderização ou afirmação de testes aprovados.
