# Issue 017 — permissões, sessões e origem

Implementação e testes em 05/11/2026. [Issue](<../../tasks/issue-017 - 017-—-Validar-permissões-sessões-origem-e-formulários.md>) · [Matriz da 016](ISSUE_016_MATRIZ.md). Aceita pelo usuário em 05/11/2026.

## Correções e regressões

- O logger HTTP propagava todos os valores de query ao logger da requisição,
  exceto nomes com `password`; o handler de busca também escrevia a query inteira.
  Uma regressão falhou com marcador fictício antes da correção. Agora método,
  caminho e status continuam nos logs, sem nomes ou valores de parâmetros.
- A primeira inicialização registrava a senha do administrador em INFO. O
  servidor real com configuração e senha fictícias reproduziu a presença antes
  da correção; após a correção o teste verifica startup e busca com log debug.
- Logout JSON com dois cookies de sessão selecionava o primeiro e podia revogá-lo.
  Uma regressão falhou com 204 antes da correção; agora ambas as ordens retornam
  400 sem revogar ou limpar qualquer sessão. Logout sem cookie continua idempotente.
- Erros de origem JSON retornavam 403 sem `Cache-Control: no-store`, pois eram
  produzidos antes de `RequireAdmin`. Regressão falhou antes da correção e passou
  após aplicar a política no ponto da rejeição.

O comando de recuperação de senha continua apresentando a senha nova na saída do
CLI, que é o canal intencional de entrega dessa credencial; ele não é log HTTP.
A configuração padrão de senha e a política de proxy não foram alteradas.

## Evidência executada

`GOCACHE=/private/tmp/koito-go-cache GOPROXY=off go build -o /private/tmp/koito-issue017 ./cmd/api`

`python3 scripts/validate_permissions_017.py --binary /private/tmp/koito-issue017`

Servidor real em loopback, SQLite v3 novo por modo, imagens externas e MusicBrainz
desativados, credenciais fictícias e diretórios removidos no fim:

| Modo | Negativas verificadas | Banco e arquivos | Segredos nos logs testados |
| --- | ---: | --- | --- |
| React/API | 498 | todas as colunas de todas as tabelas e conteúdo de arquivos inalterados; integridade/FKs OK | ausentes |
| Go | 994 | mesmas verificações; integridade/FKs OK | ausentes |

A matriz de requisições cruza famílias administrativas JSON e forms Go, métodos,
visitante, chave A/B/inválida, cookie inválido/duplicado nas duas ordens, sessão
expirada, identificador sem sessão e não-admin. São verificadas negações, no-store,
corpo sem marcadores e snapshot persistente. A chave inválida testa o mesmo
resultado de autenticação, mas não simula, por si só, revogação concorrente.
Inclui sessão admin com chave de outro perfil, origem ausente, correta, `null`,
host enganoso, esquema/porta diferentes, `Sec-Fetch-Site: cross-site`, proxy XFP,
CSRF de outra sessão, duplicado, token anterior ao reinício, GET em action e
HEAD de exportação. Snapshot usa hashes de todas as colunas, inclusive `profile_id`,
e hash de conteúdo dos arquivos persistentes; não registra valores das linhas.

Outras execuções:

- `go test ./...` passou, incluindo engine; algumas suítes estavam em cache.
- `go test -race ./engine/middleware ./internal/adminweb ./internal/adminui ./engine` passou.
- `go vet ./engine/... ./internal/adminweb/...` passou.
- Testes `engine/permissions_017_test.go` passaram com TLS real em loopback,
  erros injetados de login, lookup e logout (500 genérico, no-store e cookie
  preservado), cookies duplicados, origem e duração 24 h/30 dias comparada ao banco. Logs do servidor de teste têm apenas
  credenciais fictícias; banco de operação não foi acessado.
- `scripts/validate_admin_webui.py` passou as jornadas da 015, incluindo limite
  de login HTML/JSON nas duas ordens, multipart, duas fases, revogação por senha,
  cinco formatos de importação e integridade. Os testes unitários existentes de
  `internal/adminweb/csrf_test.go` cobrem nonce, prazo de 30 minutos com relógio
  controlado, outra sessão, restart e staging multipart sem arquivo residual.
- `engine/middleware/query_secrets_test.go` testa logger de acesso e logger de
  contexto, query múltipla/malformada e marcadores em nomes e valores.

Advisor (Sol, esforço alto): **APROVADO**, após rever correções, hashes completos,
testes de TLS e erros de armazenamento. Sua tentativa separada da suíte engine
foi bloqueada pelo sandbox ao abrir porta; a execução acima passou com acesso
local autorizado.

## Limites e próximas etapas

A contagem acima se refere às negativas do script, não é prova exaustiva de todos
os caminhos possíveis. Os testes de vida de sessão de 24 h/30 dias e expiração
exata do CSRF usam testes de store/parser ou verificação de prazo, sem esperar
horas no servidor; o reinício e token de outra sessão também foram testados no
servidor real. TLS usa `httptest` com o roteador real; falhas de store são injetadas
diretamente nos handlers e middleware, sem processo externo. Persistem para a 019 erros de lookup de perfil/chave mascarados
como 404/401, e para a 020 concorrência envolvendo revogação de requisições já
iniciadas. Os cinco defeitos de fuso permanecem na 018. A matriz global e o aceite
transversal pertencem à 021.
