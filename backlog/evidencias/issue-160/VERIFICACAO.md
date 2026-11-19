# ISSUE-160 — autorização de gestão própria e global

Data: 18/11/2026. Executor: codex, sem delegação. Branch da worktree ISSUE-160; sem push, merge, publicação ou Done.

## Implementação

`internal/admin.AuthorizeAction` concentra a decisão por capacidade e recurso: perfil próprio, conta própria, contas, catálogo e servidor. Ação desconhecida, papel desconhecido, conta não ativa, recurso ausente ou dono ausente são negados. O perfil usa `listening_profiles.user_id`, carregado pelo servidor e omitido do JSON público.

HTML e JSON autenticam a sessão em cada requisição. Os serviços de gestão também reconsultam a sessão (`AuthorizeService`); nem um objeto de usuário obtido por API key, nem um papel root mantido em memória após rebaixamento conferem autorização. A referência de sessão fica fora do JSON e não muda cookie, duração, CSRF ou tickets. Operações pessoais fixam profile_id; IDs de chave são conferidos junto ao perfil na escrita. As confirmações de exclusão pessoal revalidam a propriedade do perfil e o plano antes de aplicar a transação existente.

A criação/exclusão de contas e perfis, servidor e operações editoriais compartilhadas continuam root. As superfícies JSON de conta/perfil rejeitam campos desconhecidos, inclusive role, status e user_id; formulários conservam allowlists explícitas. Não há endpoint novo de gestão de contas, ativação ou recuperação.

A migração **006_active_admin_guards.sql** substitui os guards da v5 para contar somente administradores ativos. Migrações anteriores não foram editadas. Exclusão da dupla conta/perfil e alteração de acesso adquirem o lock de escrita antes de observar contagens. O serviço `ChangeAccountAccess` permite promover/rebaixar/desativar mediante sessão root revalidada dentro da transação. Desativar remove senha e sessões, preservando o estado pending já existente. Ativação de pending depende do fluxo de senha/token da ISSUE-162 e não foi criada aqui. Backup de v5, restauração e reinício foram incluídos nos testes; bootstrap valida o schema v6.

Decisão explícita do usuário: importar o próprio histórico pode criar entidades novas com capas e aliases do arquivo. Não concede operações editoriais sobre entidades existentes. Ver AMBIGUIDADE-IMPORTACAO.md e TestOwnerImportMetadata160.

## Inventário de rotas de gestão JSON

Prefixo `W=/apis/web/v1`; `P=W/profiles/{slug}`. V = visitante sem sessão (inclui API key sem sessão); U = usuário comum; R = root ativo. Nas linhas pessoais U só pode operar no perfil cujo user_id é o seu; outro perfil recebe 403. V recebe 401 nas rotas explícitas existentes. Limites de parsing, validação, CSRF/origem, recurso ausente ou confirmação inválida ainda podem negar uma operação autorizada.

| Métodos e rota | V | U | R | Serviço/política |
| --- | --- | --- | --- | --- |
| POST W/profiles | nega | nega | permite | CreateProfile / contas |
| PATCH P/ | nega | próprio | todos | UpdateProfile |
| GET P/management | nega | próprio | todos | AuthorizeProfile antes da contagem |
| DELETE P/ | nega | nega | permite | DeleteProfile / contas |
| PUT, DELETE P/avatar | nega | próprio | todos | SetAvatar |
| GET P/export | nega | próprio | todos | ExportHistory |
| POST P/import | nega | próprio | todos | profileImportHandler / perfil |
| GET P/relay | nega | próprio | todos | Relay |
| POST P/relay/test | nega | próprio | todos | TestRelay |
| PUT, DELETE P/relay | nega | próprio | todos | SaveRelay / RemoveRelay |
| POST P/listens | nega | próprio | todos | ManualListen |
| DELETE P/listens | nega | próprio | todos | RemovePersonalListen |
| GET, POST P/user/apikeys | nega | próprio | todos | Keys / CreateKey |
| PATCH, DELETE P/user/apikeys/{id} | nega | próprio | todos | RenameKey / RevokeKey; id+profile_id |
| GET, PATCH W/user | nega | própria conta | própria conta | conta própria / ChangeAccount |
| DELETE W/data | nega | nega | permite | PurgeData / servidor |
| DELETE, PATCH W/artist/{id} | nega | nega | permite | DeleteCatalogItem / UpdateMediaMetadata |
| DELETE, PATCH W/album/{id} | nega | nega | permite | DeleteCatalogItem / UpdateMediaMetadata |
| DELETE, PATCH W/track/{id} | nega | nega | permite | DeleteCatalogItem / UpdateMediaMetadata |
| POST W/{artist,album,track}/{id}/merge | nega | nega | permite | MergeArtists / MergeAlbums / MergeTracks |
| POST, DELETE W/{artist,album,track}/{id}/aliases | nega | nega | permite | EditAlias |
| PATCH W/{artist,album,track}/{id}/aliases/primary | nega | nega | permite | EditAlias |
| PATCH W/{artist,album}/{id}/image | nega | nega | permite | ReplaceArtwork |
| PATCH W/album/{id}/artists/{artist_id} | nega | nega | permite | SetMediaArtistPrimary |
| POST W/track/{id}/artists | nega | nega | permite | ChangeTrackArtist |
| DELETE, PATCH W/track/{id}/artists/{artist_id} | nega | nega | permite | ChangeTrackArtist / SetMediaArtistPrimary |

Aliases legados sem slug: POST/DELETE W/listens, GET/POST W/user/apikeys, PATCH/DELETE W/user/apikeys/{id} e GET W/export passam pela mesma política pessoal, depois da resolução não ambígua de perfil. Mais de um perfil continua exigindo URL explícita (409); não se seleciona outro perfil por payload.

## Inventário de rotas HTML

GET também cobre HEAD pela adaptação existente. V nunca recebe dados de gestão: alguns documentos redirecionam/exibem login, outros retornam 401; POST exige autenticação e formulários/tickets válidos.

| Métodos e rota | V | U | R |
| --- | --- | --- | --- |
| GET /admin sem seção explícita | login | mantém 403 e saída existentes | permite |
| GET /admin?section=account | login | própria conta | própria conta |
| GET /admin?section={profiles,keys,relay,import,export}&profile={id} | login | próprio | todos |
| GET /admin/settings | login | nega | permite |
| POST /admin/settings/save, /admin/settings/subsonic/test | nega | nega | permite |
| POST /admin/account/username, /admin/account/password | nega | própria conta | própria conta |
| POST /admin/profile/create | nega | nega | permite |
| GET /admin/profile/{id}/delete-review | nega | nega | permite |
| POST /admin/profile/{id}/delete | nega | nega | permite |
| GET /admin/profile/{id}/export | nega | próprio | todos |
| POST /admin/profile/{id}/{save,avatar,avatar-clear,import} | nega | próprio | todos |
| POST /admin/profile/{id}/keys/create | nega | próprio | todos |
| POST /admin/profile/{id}/keys/{key}/{label,revoke} | nega | próprio | todos |
| POST /admin/profile/{id}/relay/{save,test,remove} | nega | próprio | todos |
| GET /{slug}/manage/track/{id}/{listens,add-listen,listen} | login | próprio | todos |
| POST /{slug}/manage/track/{id}/{listen-add,listen-delete} | nega | próprio | todos |
| GET /{slug}/manage/{artist,album,track}/{id}/delete | login | próprio | todos |
| GET /{slug}/manage/{artist,album,track}/{id}/review?action=delete | login | próprio | todos |
| POST /{slug}/manage/{artist,album,track}/{id}/confirm | nega | somente plano pessoal próprio | todos; plano global exige root |
| GET /{slug}/manage/{artist,album,track}/{id}, tarefas edit/image/merge e review de merge | login | nega | permite |
| POST /{slug}/manage/{artist,album,track}/{id}/{alias-add,alias-primary,alias-delete,metadata,artist-add,artist-remove,primary-artist,artwork} | nega | nega | permite |
| POST /admin/login | preauth existente | preauth existente | preauth existente |
| POST /admin/logout | sem sessão válida nega | sessão atual | sessão atual |

IsRoute/IsPage/IsAction continuam como allowlist estrutural, com capacidade explícita antes de parsing/estágio. Rota/tarefa não listada não herda autorização, mesmo para root. Editor global não foi liberado como atalho para dados pessoais; os documentos pessoais existentes têm seus próprios caminhos.

## Fronteiras preservadas

- GET W/profiles e GET P/ seguem públicos, com identidade pública sem user_id de proprietário. Leituras de histórico em `bindProfileReads` (artist/album/track, aliases, artistas relacionados, interest, top, listens, atividade, primeiro evento, now-playing, stats, summary e search), documentos públicos, imagens e caches não receberam gate de require_login: isso é ISSUE-161, explicitamente fora do escopo.
- POST /apis/listenbrainz/1/submit-listens e GET /apis/listenbrainz/1/validate-token conservam o middleware de ingestão. Chave resolve o destino, não um ator humano; não dá acesso a chaves, relay, exportação ou gestão. Não se afirma cobertura de privacidade das escutas/imagens/cache que depende da ISSUE-161.
- /config, /health, /login, /logout JSON, /language e assets conservam seus contratos existentes. Sem novos limites de login, sem mudança de bcrypt, hash fixo, admin/changeme, CSRF, quota de tickets ou formato de sessão.
- Telas/navegação por permissões e destino padrão do usuário comum continuam em ISSUE-163/157. Campos de visibilidade e gestão root por token continuam nas etapas 161/162.

## Testes e critérios

- `TestManagementPolicy160`: capacidades e negação por padrão (visitante, dono, outro usuário, root, pending e papel desconhecido).
- `TestPersonalServices160` e `TestServicesRequireHumanSession160`: chamadas diretas, recursos de terceiros, API key atribuída a root, sessão revogada e papel root obsoleto; serviços de import/export, chaves, perfil/avatar, relay, escutas e catálogo.
- `TestJSONManagementMatrix160`: inventário pessoal JSON, visitantes, API keys, dono, root, catálogo global e rotas inexistentes. Validação de payload pode retornar 400/404 depois de autorizar; casos de sucesso também executam escritas pessoais reais.
- `TestMassAssignmentAndCurrentRole160`: JSON role/status/user_id/id rejeitados, ID de chave alheia, payload escolhendo outro profile_id e perda imediata de privilégios globais após rebaixamento.
- `TestHTMLManagementMatrix160`: formulários reais com CSRF/tickets, escopos próprios/alheios, campos extras rejeitados, tarefas pessoais/editoriais, exclusão individual e por plano, outro histórico e catálogo preservados, tarefas desconhecidas negadas.
- `TestOwnerImportMetadata160`: usuário comum cria entidades novas com capas/aliases e não edita metadados das entidades já existentes ao reimportar; perfil alheio negado.
- `TestLastActiveAdministrator160`: duas conexões SQLite concorrentes, com admin pendente adicional, para rebaixamento, desativação, exclusão e exclusão concorrente com desativação. Exatamente uma operação vence e resta um admin ativo.
- `TestAccessChangesUseCurrentSession160`: promoção/rebaixamento permitidos somente ao root atual; usuário não se promove e sessão rebaixada/expirada não altera acesso.
- `TestActiveGuardUpgrade160` e `TestAuthorizationBackup160`: upgrade v5, triggers contra SQL direto, backup consistente, restauração, cardinalidade e reinício.

ACs de gestão #1/#2/#3 avaliados dentro das fronteiras explícitas da ISSUE-160: não incluem a implementação futura de gate de leitura/visibilidade, tokens ou navegação. O critério sobre chave de ingestão comprova ausência de sessão humana e negação dos dados privados de gestão; não substitui os testes do gate da ISSUE-161.

## Resultado final

- **make check: PASSOU**, incluindo go vet, testes Go sem cache, verificações UI/referências e **13/13 jornadas E2E** (web-assets, multiuser, admin, public, permissions, isolation, migration, ingest-migration, standalone, legacy-env, shutdown, vscode e tools).
- **go test -race -count=1 ./engine/... ./internal/admin ./internal/adminui ./internal/adminweb ./internal/db/sqlite ./internal/models -timeout 10m: PASSOU**. Engine 168,496s; SQLite 123,848s; demais pacotes também verdes, sem data race.
- **go test -race -count=1 ./engine -run 160 -timeout 120s: PASSOU** (12,286s), repetindo os últimos casos acrescentados de exclusão pessoal por plano e IDs de chave/destino manipulados.
- **go test -race -count=1 ./engine/middleware -timeout 120s: PASSOU** (1,727s), após preservar o log interno de falhas de sessão na revisão final.
- **git diff --check: PASSOU**.
- **Graphify AST atualizado a partir desta worktree** pelo script canônico com o caminho da worktree como argumento: 4249 nós, 18953 arestas. Consulta posterior localizou política, serviços e adaptadores ligados. Nenhuma extração semântica/LLM executada; custo de API: zero tokens.
- Revisão local do diff e deslop executadas pelo próprio codex; parecer em REVIEWER.md. Não houve nova delegação.

Rodadas intermediárias encontraram expectativas anteriores de schema v5 e bloqueio da própria conta; foram alinhadas ao schema v6 e à política nova. Fixtures de imagem que usavam usuários sem sessão passaram a autenticar pela mesma fronteira dos handlers, sem enfraquecer os guards. Uma rodada inicial de race com timeout de 120s expirou; a rodada completa com 10m acima passou. Nada disso permanece como falha aberta.

Os três ACs de gestão foram comprovados nas superfícies existentes da ISSUE-160, conforme o escopo descrito acima. Não houve aceite do usuário sobre a entrega de código, mudança para Done ou autorização de publicação.
