# Issue 016 — matriz de contratos e fixture

Leitura e verificação da fixture em 05/11/2026. [Issue 016](<../../tasks/issue-016 - 016-—-Contratos-fixture-e-matriz-de-validação-transversal.md>) · [Roadmap 5](<../../docs/roadmaps/doc-008 - Roadmap-5-—-comportamento-permissões-fusos-e-isolamento-entre-perfis.md>).

## Como ler a matriz

**E** = teste existente identificado por leitura, sem afirmar execução nesta issue;
**L** = lacuna a executar; **D** = decisão pendente; **F** = defeito reproduzido
no estudo anterior. Testes anteriores são referências parciais, não aprovação
da combinação completa de ator, método, perfil, modo e estado. Os status abaixo
são contratos esperados ou comportamento observado por leitura, salvo indicação.
Cobertura real de cada linha ainda deve ser medida nas issues indicadas.

Atores: V visitante; A sessão admin válida; N sessão não-admin; I sessão ausente,
inválida, expirada ou revogada; K-A/K-B chave de ingestão; K-X chave inválida ou
revogada. Em leitura pública, incluir todos esses atores, inclusive cookie e chave
conflitantes: perfil é público e credenciais não devem modificar o escopo da URL.
M = modo React/API ou modo Go; JSON e ListenBrainz são comuns aos dois modos.
S = slug existente, X = inexistente; 0/1/N = quantidade de perfis. Em todas as
negativas verificar resposta **e** que dados, arquivos e cache não mudaram.

## Inventário e matriz por família

As fontes de rotas são `engine/routes.go`, `internal/webui/routing.go`,
`internal/adminui/routing.go`; manifestos existentes: `test_assets/web-ui-routes.json`
e `test_assets/admin-web-ui-routes.json`, usados por testes Go e TypeScript.
A coluna Observado é leitura de código; os resultados finais dependem do caso.
Nas linhas JSON após `/apis/web/v1/profiles/{slug}`, sufixos como `/relay`,
`/management`, `/import` e `/listens` são relativos a esse prefixo; as leituras
legadas têm prefixo `/apis/web/v1` e não incluem `/profiles/{slug}`.

| Família/operação e método | Ator, perfil, modo e contrato esperado | Observado e cobertura/lacuna | Responsável |
| --- | --- | --- | --- |
| GET/HEAD `/`, `/{slug}`, charts, listens, rewind, artist/album/track | V/A/N/I/K em S, M: documento público no perfil da URL; X não expõe outro perfil | E: `webui/handlers_test.go`, `media_test.go`, `rewind_test.go`; L: cruzamento completo atores e zero/um/vários | 019/021 |
| GET/HEAD `/{slug}/search` | V/A/N/I/K em S, modo Go: documento de busca pessoal | React usa busca na interface e JSON; `client/app/routes.ts` não registra documento `/{slug}/search`; L: cobertura por modo e escopo | 019/021 |
| GET/HEAD documentos legados `/chart/...`, `/listens`, `/artist/...` | V, M=Go: com 1 perfil redirecionar para ele; 0 ou N para seletor com retorno seguro | E: `TestRoutingSearchAndLegacy`; L: 0/1/N, URL inválida e falha store | 019 |
| GET/HEAD `/admin`, `/admin/profile/{id}/export`, `/delete-review`, `/{slug}/manage/{kind}/{id}[/review]` | A: documento/download ou 404 válido; V/I: login/negação; N: negar; K: não autentica; Go | E: `TestAdministrationJourney`, `validate_admin_webui.py`; L: HEAD e matriz de credenciais/recursos ausentes | 017 |
| POST `/admin/login`, `/logout`, `/account/{username,password}` | Login com nonce pré-auth; logout/conta vinculados à sessão; Go | E: `TestLoginCSRFNonce`, `TestAdminSessionCookiesAndPrivileges`; L: expiração, cookies duplicados, 24 h/30 d, logout e duas abas | 017 |
| POST `/admin/profile/create`, `/admin/profile/{id}/{save,avatar,avatar-clear,delete,import}`, `/admin/profile/{id}/keys/{create,{keyID}/label,{keyID}/revoke}`, `/admin/profile/{id}/relay/{save,test,remove}` | A + CSRF/ticket válido: perfil explícito; I/N/K/CSRF inválido: negar; Go | E: `validate_admin_webui.py`, `TestOneUseSubmission`; L: cada método/ator e upload antes da validação; criação/alteração sujeitas à validação própria | 017/019/020 |
| POST `/{slug}/manage/{kind}/{id}/{alias-add,alias-primary,alias-delete,metadata,artist-add,artist-remove,primary-artist,listen-add,listen-delete,artwork,confirm}` | A + CSRF/ticket: mídia global; escuta pessoal do slug; outros atores negar; Go | E: `TestPreparedMediaFormsAndProfileContext`, `TestCatalogReviewStaleAndOneUse`; L: combinações de tipo/ação, ator, método e perfil | 017/019/020 |
| GET `/apis/web/v1/config`, `/profiles`, `/health` | V/A/N/I/K, M: públicos; health reflete prontidão | E: `engine/long_test.go`, validadores; L: perfis vazios, cache e métodos inesperados | 017/021 |
| GET `/apis/web/v1/profiles/{slug}` e leituras `/artist/{id}[/aliases|interest]`, `/album/{id}[/artists|aliases|interest]`, `/track/{id}[/artists|aliases|interest]`, `/top/{artists|albums|tracks}`, `/listens`, `/listen-activity`, `/first-activity`, `/now-playing`, `/stats`, `/search`, `/summary` | V/A/N/I/K, S, M: dados pessoais de S; X = 404 observado; item exclusivo de outro perfil não aparece | E: `TestProfileBackendIsolationAndIngestion`, `TestMediaHeadersMatchAPIAndKeepProfileScope`, `TestInterestSQLiteScopesEntitiesAndExcludesFuture`; L: matriz por família, Empty, relações e falha store | 019 |
| GET leituras JSON legadas `/apis/web/v1/{mesmas leituras}` | V/A/N/I/K: 1 perfil resolve; 0 ou N = 409 observado | E: `validate_multiuser.py` cobre N; L: zero, todos atores e falha de listagem = 500 observado | 019 |
| POST `/apis/web/v1/login`, `/logout`; GET/PATCH `/user` | JSON login válido = 204 observado; GET/PATCH user exige A; logout limpa cookie; N = 403 em `RequireAdmin` | E: `TestAdminSessionCookiesAndPrivileges`; L: origem, limites e combinações de sessão | 017 |
| POST `/profiles`; PATCH/DELETE `/profiles/{slug}`; GET `/management`; PUT/DELETE `/avatar` | A: operações em S; V/I/K sem sessão negados; N = 403 se S; M | Observado: `ProfileScope` antes de `RequireAdmin`; X pode retornar 404 antes de 401. E: `TestProfileManagementLifecycle`; L: diferenças de precedência e estado após falha | 017/019 |
| GET/POST/PATCH/DELETE `/profiles/{slug}/user/apikeys[/{id}]`; GET/PUT/DELETE `/relay`, POST `/relay/test` | A apenas; chave/corpo estrangeiro não transfere propriedade; testar cross-profile, no-store e relay local | E: `TestProfileRelayAdministration`, `TestProfileKeysCannotMoveAndRelayDoesNotReassign`; L: matriz completa e erros de rede | 017/019/020 |
| GET `/profiles/{slug}/export`; POST `/profiles/{slug}/import`; POST/DELETE `/profiles/{slug}/listens` | A em S; export pessoal; import no destino; escuta identificada por perfil/faixa/tempo | E: `TestAdminUIImportExport`, `TestDeleteListenRequiresExactProfileAndRecord`; L: múltiplos atores, HEAD, concorrência e falhas | 017/019/020 |
| DELETE/POST/PATCH `/artist/{id}`, `/album/{id}`, `/track/{id}` e aliases, merge, image, artists | A apenas, catálogo compartilhado global; N/I/K negados; impacto sobre perfis é deliberado | E: `TestAdminMergeAPIContracts`, `TestCatalogPlanStaleAndScopedMerge`; L: matriz completa dos métodos e impacto entre pessoas | 017/019/020 |
| Rotas administrativas legadas `/listens`, `/user/apikeys[/{id}]`, `/export` | A + exatamente um perfil; sem A = 401 antes da seleção; A com 0/N = 409 observado | E: `validate_multiuser.py`, `TestAdminSessionCookiesAndPrivileges` parcialmente; L: precedência 0/1/N | 017/019 |
| DELETE `/apis/web/v1/data` | Somente A; limpeza global de escutas e catálogo, em banco descartável | E: `TestDelete`; L: matriz de credenciais e impacto global explícito | 017/020 |
| POST `/apis/listenbrainz/1/submit-listens`, GET `/validate-token` | K-A grava A, K-B grava B; K-X/V = 401; cookie/corpo não substituem chave; M | E: `TestProfileBackendIsolationAndIngestion`, `validate_multiuser.py`; L: revogação concorrente, corpo de tamanho excessivo | 019/020 |
| GET/HEAD `/image/{id}/{filename}`, `/public/*`, `/web-assets/*`, assets React; método inesperado | V, M: recursos conforme modo, sem mutação; HEAD protege recursos que exigem autenticação | E: manifestos, `TestPreviewRoutesAndEscaping`; L: método/status/Content-Type/dispatcher e ausência de mutação; fallback SPA pode responder HTML | 017/021 |

`chi.GetHead` executa a cadeia de GET para HEAD; observar headers, autenticação,
estado e corpo. Não impor 404/405 universal: fallback e curingas têm comportamento
próprio. A busca global via `WithoutProfile` é capacidade de gestão autenticada
Go; não existe contrato de novo endpoint JSON público global. Mídia pública exige
relação pessoal; `playing_now` de faixa ainda não escutada é caso separado.

### Precedência, erro e políticas transversais

| Caso | Esperado / observado por leitura | Evidência e estado | Issue |
| --- | --- | --- | --- |
| Perfil explícito X + visitante em JSON administrativo | X resolvido antes da sessão: 404 pode preceder 401; rotas legadas admin autenticam primeiro | `engine/routes.go`, `ProfileScope`; L: servidor real com mesmas entradas | 017 |
| Store de perfil falha | Desejado: erro de infraestrutura distinto de ausência; atual: lookup slug retorna 404 para qualquer erro | `engine/middleware/profiles.go`; L: falha injetada e decisão de correção | 019 |
| Store de chave falha | Desejado: distinguir erro operacional de chave inválida; atual: lookup transforma qualquer erro em 401 | `IngestionKey`; L: falha injetada sem vazar segredo | 019 |
| Sessão, origem, cookies, preauth e multipart | HTML exige CSRF; JSON preserva contrato SameOrigin; sem segredo em URL/log/cache | E: `adminweb/csrf_test.go`, `admin_contract_test.go`; L: matriz por ator/método e proxy | 017 |
| Ações repetidas e planos obsoletos | Uma mutação por ticket; conflito na revalidação; tickets locais ao processo | E: `TestOneUseSubmission`, `TestCatalogReviewStaleAndOneUse`; L: reinício/concorrência | 017/020 |
| Requisição já autorizada vs revogação | D: definir fronteira; não presumir cancelamento retroativo | Leitura de código; L: interleavings determinísticos | 020 |
| `now-playing` | Cache por perfil; invalidar A preserva B; item sem escuta tem tratamento próprio | E: `TestNowPlayingIsScopedAndMissingTrackClearsOnlyItsProfile`; L: chaves conflitantes/merge/reinício | 019 |
| Export >1000, empates | Cursor crescente `(listened_at, track_id)`; sem omissões/repetições no cenário estático; export dinâmico sem snapshot | E: `TestProfileExportPaginationDoesNotSkipSameTimestamp`; L: fronteira de página desta fixture e concorrência | 019/020 |
| Cinco casos de fuso | Resultados corretos nos oráculos de [reprodução](reproduce_time_results.txt) | F: store/helper reais retornaram resultado incorreto; não são regressões permanentes | 018 |
| Clock/timezone | ForceTZ > query válida > cookie válido > fuso do processo; relógios de sessão e consultas controlados nos testes que exigem fronteira | E: `TestHTMLContracts`, `TestRewindForcedTimezoneOverridesQueryAndCookie`; L: relógio de autorização e da aplicação não é injetado por esta fixture | 017/018 |

## Oráculos independentes da fixture

`scripts/roadmap05_fixture.py` cria uma base nova em `TemporaryDirectory`, aplica
as migrations 001–003 em SQLite e semeia apenas dados fictícios; `disposable_fixture()`
permite às próximas issues usar a base enquanto o contexto estiver aberto. O
programa recusa preencher conexão com schema existente. Imagens são IDs simbólicos
compartilhados, sem arquivos de imagem; testes de I/O devem criar seus próprios
arquivos temporários. As chaves são fictícias e não há conta/sessão criada: testes
HTTP devem preparar autenticação isolada e observar o trigger de usuário legado.

Tempo de referência: `2025-06-01T12:00:00Z`, fuso de referência UTC; timestamps
da fixture são fixos. **A fixture não altera o fuso do processo nem injeta relógio**
no servidor. Cenários relativos ao relógio exigem controle explícito ou períodos
absolutos, e testes de expiração exigem relógio separado na 017.

| Identidade | Catálogo/escutas fixas | Resultado independente esperado |
| --- | --- | --- |
| Alpha (1) | faixa comum 1 em `2024-05-01T00:00:00Z` e +1 s; faixa 4 em +2 s | 3 escutas; faixas 1=2 e 4=1; álbum 1=3; artistas 1=3 e 2=2; duração total 560 s |
| Beta (2) | faixa comum 1 no mesmo instante de Alpha; faixa exclusiva 2 em +3 s, duração 0 | 2 escutas; faixas 1=1 e 2=1; álbuns 1=1 e 2=1; artistas 1=1 e 2=2; duração total 180 s |
| Empty (3) | sem escutas | 0 escutas e nenhum item pessoal ouvido |
| Catálogo | faixa 3 sem escutas; artista 1 com alias `共通`; artista 2 também associado à faixa comum; imagem simbólica comum | faixa 3 não entra em consultas pessoais; duas pessoas podem compartilhar faixa/instante; contagens de artistas podem somar mais que escutas |

A tentativa duplicada `(Alpha, faixa 1, instante base)` é recusada pela constraint
SQL na checagem; a ingestão real usa `ON CONFLICT DO NOTHING`, não equivale a
uma exceção HTTP. O esperado é uma linha pessoal, duas linhas entre Alpha e Beta;
verificar também o número de tentativas e resultado da API na 019.

Variante grande adiciona 1002 escutas em Alpha, alternando faixas 1/4 no mesmo
segundo: Alpha 1005, Beta 2, Empty 0. O empate cruza a posição 1000/1001 no
cursor de export crescente; a validação da fixture só confirma dados/ordem, não
executa exportação. Mutações futuras usam cópia própria e checam `integrity_check`,
`foreign_key_check`, contagens por perfil e IDs sobreviventes após cada cenário.

## Parsing e estatísticas: observado, sem impor uniformização

| Entrada/campo | HTML Go | JSON legado / situação | Próximo passo |
| --- | --- | --- | --- |
| `limit=0/501/inválido`, `page` inválida | HTML rejeita limit inválido; page 0 normaliza 1, negativo/overflow rejeita | `OptsFromRequest` usa 100 e page 1 quando inválidos; leitura | `TestHTMLContracts`; testar por endpoint na 018 |
| `year/month/week`, from/to conflitantes | HTML valida intervalo/semana ISO e rejeita conflitos | parser legado ignora erro numérico e mantém valores zero; comportamento posterior depende do handler | `webquery/html.go`, `legacy.go`; 018 |
| Parâmetros repetidos | HTML geral usa `Query.Get` (primeiro); rewind exige exatamente um | parser legado usa `Query.Get`; não declarar política uniforme | `webquery/rewind_test.go`; 018 decide contrato |
| Período ausente | HTML home usa week; listagens usam all_time | parser JSON padrão all_time; alguns handlers podem decidir defaults próprios | consultar endpoint antes de comparar; 018 |
| Contagens/duração em `ReadStats` | Recebem timeframe | `DaysActive` e `LongestStreak` recebem somente fuso; média divide contagem do período por dias ativos sem timeframe | D: definir semântica; 018. Não tratar mistura como resultado normativo |
| Summary | Intervalo fixado no serviço; calendário usa dias civis; janela móvel usa períodos de 24 h completos para média | E: `internal/summary/timeframe_test.go`; limites SQL existentes preservados | 018 verifica oráculo por dados e DST |

Nenhum status acima considera todos os endpoints de forma indistinta; os casos
malformados e o comportamento de HTTP devem ser medidos individualmente.
