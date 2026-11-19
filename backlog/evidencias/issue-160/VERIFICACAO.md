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

## Verificação inicial (18/11/2026)

- **make check: PASSOU**, incluindo go vet, testes Go sem cache, verificações UI/referências e **13/13 jornadas E2E** (web-assets, multiuser, admin, public, permissions, isolation, migration, ingest-migration, standalone, legacy-env, shutdown, vscode e tools).
- **go test -race -count=1 ./engine/... ./internal/admin ./internal/adminui ./internal/adminweb ./internal/db/sqlite ./internal/models -timeout 10m: PASSOU**. Engine 168,496s; SQLite 123,848s; demais pacotes também verdes, sem data race.
- **go test -race -count=1 ./engine -run 160 -timeout 120s: PASSOU** (12,286s), repetindo os últimos casos acrescentados de exclusão pessoal por plano e IDs de chave/destino manipulados.
- **go test -race -count=1 ./engine/middleware -timeout 120s: PASSOU** (1,727s), após preservar o log interno de falhas de sessão na revisão final.
- **git diff --check: PASSOU**.
- **Graphify AST atualizado a partir desta worktree** pelo script canônico com o caminho da worktree como argumento: 4249 nós, 18953 arestas. Consulta posterior localizou política, serviços e adaptadores ligados. Nenhuma extração semântica/LLM executada; custo de API: zero tokens.
- Revisão local do diff e deslop executadas pelo próprio codex; parecer em REVIEWER.md. Não houve nova delegação.

Rodadas intermediárias encontraram expectativas anteriores de schema v5 e bloqueio da própria conta; foram alinhadas ao schema v6 e à política nova. Fixtures de imagem que usavam usuários sem sessão passaram a autenticar pela mesma fronteira dos handlers, sem enfraquecer os guards. Uma rodada inicial de race com timeout de 120s expirou; a rodada completa com 10m acima passou. Nada disso permanece como falha aberta.

Os três ACs de gestão foram comprovados nas superfícies existentes da ISSUE-160, conforme o escopo descrito acima. Não houve aceite do usuário sobre a entrega de código, mudança para Done ou autorização de publicação.


## Correções após o parecer do Reviewer — 19/11/2026

O parecer externo original e o comentário do Reviewer foram preservados, incluindo as ressalvas históricas: `PARECER-REVIEWER.md` e comentário de autoria reviewer na tarefa. O usuário reafirmou a opção 2; a confirmação foi registrada com autoria **ebrgr** na tarefa e incorporada ao **doc-041 §2**, pelo comando `backlog doc update doc-041 --content ...`. Essa decisão mantém o recurso, sem representar aceite da entrega ou publicação.

### A — SSRF das imagens importadas

`imagefetch.PublicSource` grava o fragmento reservado `#tunetrail-public-only` na URL de imagem vinda de importação Koito/TuneTrail por não-root. A marca persiste no banco, no export e no reimport; não exige coluna ou migração nova. Os adaptadores JSON (arquivo do diretório e upload multipart) e HTML passam a origem autenticada ao importador; o corpo/arquivo não escolhe esse privilégio. O serviço reconsulta a sessão para decidir a origem, sem confiar num papel armazenado no objeto do chamador.

Na busca efetiva, `imagefetch.request` reconhece a marca e remove o fragmento antes da requisição. O transporte restrito não usa proxy de ambiente; resolve DNS, rejeita conjuntos de respostas com IP proibido, conecta ao literal aprovado e confere o IP efetivamente conectado. Cada conexão de redirect passa pela mesma política. HTTP(S) somente, sem credenciais, no máximo três redirects seguidos, timeout total de 10s, limites de dial/headers/TLS e corpo de até 10 MiB. Loopback, RFC1918, ULA, link-local, unspecified, metadata de nuvem, CGNAT, multicast, endereços de transição IPv6 e IPv4 mapeados internos são negados. Fontes do root sem a marca continuam aceitando servidores em rede local; não foi imposto bloqueio global às imagens do root.

Testes:

- `TestPublicAddress160`: IPv4/IPv6 proibidos, metadata, mapeamentos e endereços públicos.
- `TestImportedImageLocalBlock160`: marca persistente e idempotente, Download não alcança o servidor local marcado; fonte root sem marca continua funcionando; esquemas/credenciais inválidos rejeitados.
- `TestPublicDNSAndConnectedIP160`: DNS privado e misto rejeitados antes de dial, DNS resolvido uma vez e conexão ao literal público, peer privado ou divergente rejeitado.
- `TestPublicRedirects160`: primeiro hop público simulado sem internet; redirects para IP privado e nome que resolve privado bloqueados sem conexão ao destino; loop limitado, deadline/cancelamento e resposta pública positiva.
- `TestOwnerHTTPImportProvenance160`: os três caminhos humanos de importação preservam a marca; a requisição anônima posterior de imagem em `/image/{id}/128x128.webp` retorna fallback e **zero requisições ao serviço interno**. A importação não tenta remover as imagens do recurso.
- `TestDownloadBoundsAndCancellation` continua cobrindo corpo acima do limite e cancelamento no leitor comum de Download, usado também pelas fontes restritas.

### B — Pré-semeadura do catálogo

Mitigação implementada **sem mudança de schema**. `source` dos aliases de entidades novas criadas por import não-root recebe `ProfileImport:<profile_id>`, inclusive o alias canônico e aliases extras. Nomes, imagens e aliases continuam exibidos no histórico e exportados. A resolução implícita por nome/alias em artistas, álbuns e faixas ignora essa origem em outros destinos. `CatalogProfile` preserva a identidade do destino quando `WithoutProfile` remove somente o escopo de leitura das escutas; não cria um gate de visibilidade da ISSUE-161.

MBIDs desconhecidos não são gravados nas entidades novas do import não-root. MBIDs já existentes continuam resolvendo o catálogo sem editar suas entidades; quando um MBID desconhecido foi descartado, a repetição do import usa o nome do próprio perfil e evita duplicações. O enriquecimento automático futuro também conserva a proveniência dos aliases e não preenche MBIDs que promoveriam a entidade ao namespace global. Somente uma operação editorial explicitamente autorizada ao root marca o contexto de validação e pode promover MBIDs/aliases. Importações de operador/root e ingestão ListenBrainz conservam seus contratos; essa mitigação não é uma reforma geral da identidade musical ou do protocolo de ingestão.

Testes:

- `TestOwnerImportResolution160`: mantém resolução própria; nega resolução por nome/aliases e MBID desconhecido em outro destino; cobre títulos alternativos de álbum e informações de faixa; impede promoção automática de MBIDs e aliases; permite que uma entidade legítima posterior use o nome/MBID e verifica validação editorial explícita por root.
- `TestImportProfileIdentity160`: excluir a conta/perfil e cadastrar outra não reutiliza a identidade do perfil; os aliases antigos não passam a resolver para a conta substituta, enquanto os metadados do catálogo continuam preservados.
- `TestOwnerHTTPImportProvenance160`: arquivos com MBIDs desconhecidos não os gravam nas três tabelas, gravam a proveniência correta e continuam importando escutas; repetir o upload com esses MBIDs não duplica artistas/álbuns/faixas/escutas.
- `TestOwnerImportMetadata160`: imagens e aliases extras continuam presentes, com a marca de rede na imagem; reimportar não edita metadados existentes.

Consequência documentada no doc-041: nomes iguais podem representar entidades distintas em destinos diferentes até reconciliação explícita pelo root. Não se certifica a autenticidade semântica de metadados de arquivos, e não se cria sobreposição de catálogo por usuário ou uma nova tabela de identidade. Os registros compartilham as tabelas existentes; a resolução implícita usa a proveniência persistida.

### C — Retorno HTML do dono

`authorizePage` interpreta ausência de section com profile presente como seção profiles. O redirect existente `/admin?profile=N` passa pela autorização pessoal e exibe o aviso de sucesso; `/admin` sem perfil continua com o destino anterior, sem antecipar a navegação da ISSUE-163.

`TestHTMLManagementMatrix160` segue os redirects de salvar perfil, criar chave, remover relay e limpar avatar; exige HTTP 200 e o aviso “Changes saved.”. O upload HTML em `TestOwnerHTTPImportProvenance160` também segue o redirect e comprova o resumo “Imported: 1”. CSRF e a ordem dos campos multipart da 159 foram preservados.

### D — Último admin com WAL

As duas conexões concorrentes de `TestLastActiveAdministrator160` usam `_pragma=journal_mode(WAL)` e o teste confere `PRAGMA journal_mode = wal`. Permanecem os cenários de rebaixamento, desativação, exclusão e operação mista, com administrador pending adicional e exatamente um administrador ativo sobrevivente. **Migração 006 intacta; nenhuma 007 criada.**

### E — Campos extras (opcional)

A tolerância em post_listens, apikeys e profile_relay foi mantida para não mudar contratos/clientes sem necessidade. O destino segue resolvido pelo servidor; um profile_id extra não é aplicado. Permanecem a rejeição estrita em conta/perfil e as allowlists HTML. A API ListenBrainz não mudou. O teste adversarial anterior de profile_id no corpo continua provando que nenhuma escuta vai para outro perfil.

### Validação das correções

- `make check`: **PASSOU**, com go vet, todos os testes Go sem cache, UI/referências e **13/13 jornadas E2E**.
- `CGO_ENABLED=1 go test -race -count=1 ./engine/... ./internal/admin ./internal/adminui ./internal/db/... ./internal/imagefetch ./internal/importer ./imagecache -timeout 10m`: **PASSOU**, sem data race. Engine 171,039s, SQLite 122,392s; demais pacotes verdes.
- `CGO_ENABLED=1 go test -race -count=1 ./engine ./internal/admin ./internal/imagefetch -run '160|TestPublic|TestImported' -timeout 3m`: **PASSOU**, após os últimos ajustes e o caso de upload repetido com MBIDs desconhecidos (engine 20,092s).
- `git diff --check`: **PASSOU**.
- Graphify AST atualizado a partir desta worktree: **4297 nós e 19137 arestas**, sem extração semântica/LLM; custo de API zero tokens.
- Revisão diferencial/deslop pelo próprio executor, sem delegação: `REVISAO-CORRECOES.md`. Parecer original preservado em `PARECER-REVIEWER.md`; não se atribui ao Reviewer externo uma nova revisão destas correções.

A política central, o schema v6, a 159 e os limites de login permanecem preservados. Tarefa **Em andamento**, sem push, merge, publish ou Done.
