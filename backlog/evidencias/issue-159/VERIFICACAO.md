# ISSUE-159 — Autenticação e sessões

Execução direta pelo Codex na worktree da tarefa, sem re-delegação. Conferidos AGENTS.md/CLAUDE.md, doc-041, decision-010, os planos `01a1ea88-202b-7aa1-ad3b-59be658fbb6b` e `01a1efec-ad9e-77f3-b021-068cc05ae40a` e a verificação da ISSUE-158.

## Alterações e limites

- `admin.Login` aceita contas active dos dois papéis, com o bcrypt e validação existentes. `Authenticate` resolve a conta ativa sem autorizar administração. `SaveAuthenticatedSession` remove somente o filtro de papel: a condição atômica por ID, hash atual e status active permanece.
- `AdminIdentity` combina sessão válida e papel atual para os consumidores HTML antes dependentes da autorização implícita de `SessionIdentity`. Páginas administrativas, configuração, revisão/exclusão, exportação e gestão de mídia continuam restritas a admin. A navegação pública conserva sua verificação administrativa. `RequireAdmin` e as autorizações dos serviços permanecem intactos.
- CSRF vincula formulários à sessão autenticada, sem exigir admin. Tickets já usavam UUID de sessão e não precisaram mudar de formato ou armazenamento; os testes comprovam uso por contas comuns, rejeição entre contas e replay.
- Duração 24h/30 dias, cookie, SameOrigin, preauth, parsing e política de senhas preservados. Nenhum limitador novo; `engine/routes.go` não mudou.
- A gestão própria (inclusive autorização de `ChangeAccount` e suas rotas), visibilidade, ativação/reset por token e telas multiusuário continuam nas ISSUE-160–163. Nesta issue, a revogação de sessões comuns na troca de senha é exercitada pelo `UpdateUser` existente; não foi aberto endpoint de gestão ao usuário comum. O login HTML conserva seu redirecionamento existente, e o destino administrativo nega o usuário comum.

## Critérios comprovados

1. Login user/admin, sessões por conta e isolamento: `TestUserSessionsAndPasswordIsolation`, `TestUserAuthenticationKeepsAdminRoutesClosed159` e testes existentes de duração/expiração/logout. O papel atual é relido após rebaixamento, sem invalidar a autenticação nem conservar autorização admin. Contas pending continuam recusadas em `accounts_test.go`.
2. CSRF comum: suíte `csrf_test.go` exercitada com identidade user, incluindo sessão diferente, ausência de identidade, expiração, reinício, propósito, multipart e limites. `TestUserSubmissionIsolation159` usa duas contas comuns reais e seus logins. Troca de senha revoga todas as sessões da conta, preserva a sessão da outra conta e bloqueia login antigo pausado após bcrypt; a corrida admin e rollback de revogação também permanecem cobertos.
3. `TestSharedLoginLimiter159` esgota as dez tentativas alternando HTML/JSON e comprova que outro nome e outro endereço não criam orçamento separado. O limitador/configuração originais foram preservados integralmente.

## Validação

- `make check`: passou (Go vet/test, verificações de UI/JavaScript/estilo, referência visual aceita e 13/13 jornadas e2e).
- `CGO_ENABLED=1 TUNETRAIL_WEB_DIR="$PWD/assets" go test -race ./internal/admin ./internal/adminweb ./internal/adminui ./internal/db/sqlite ./internal/webui ./engine`: passou em todos os seis pacotes, incluindo as corridas de login/troca de senha.
- Graphify atualizado por AST a partir desta worktree e consultado para Authenticate, AdminIdentity, sessionBinding e SaveAuthenticatedSession; sem extração semântica.
- `git diff --check`: limpo. Revisão própria focada na fronteira autenticação/autorização e no diff, conforme Go/deslop. Não constitui parecer de outro agente; o responsável pelo despacho pode executar a revisão posterior.
- A primeira tentativa de `go test` direto não fornecia TUNETRAIL_WEB_DIR e falhou ao localizar assets. A execução oficial via Makefile usa o diretório correto.

Sem push, merge, publish ou Done. Status mantido Em andamento.
