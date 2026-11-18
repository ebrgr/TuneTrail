# ISSUE-159 — Autenticação e sessões

Execução direta pelo Codex na worktree da tarefa, sem re-delegação. Conferidos AGENTS.md/CLAUDE.md, doc-041, decision-010, os planos `01a1ea88-202b-7aa1-ad3b-59be658fbb6b` e `01a1efec-ad9e-77f3-b021-068cc05ae40a` e a verificação da ISSUE-158.

## Alterações e limites

- `admin.Login` aceita contas active dos dois papéis, com o bcrypt e validação existentes. `Authenticate` resolve a conta ativa sem autorizar administração. `SaveAuthenticatedSession` remove somente o filtro de papel: a condição atômica por ID, hash atual e status active permanece.
- `AdminIdentity` combina sessão válida e papel atual para os consumidores HTML antes dependentes da autorização implícita de `SessionIdentity`. Páginas administrativas, configuração, revisão/exclusão, exportação e gestão de mídia continuam restritas a admin. A navegação pública conserva sua verificação administrativa. `RequireAdmin` e as autorizações dos serviços permanecem intactos.
- CSRF vincula formulários à sessão autenticada, sem exigir admin. Tickets já usavam UUID de sessão e não precisaram mudar de formato ou armazenamento; os testes comprovam uso por contas comuns, rejeição entre contas e replay.
- Duração 24h/30 dias, cookie, SameOrigin, preauth, parsing e política de senhas preservados. Nenhum limitador novo; `engine/routes.go` não mudou.
- A gestão própria (inclusive autorização de `ChangeAccount` e suas rotas), visibilidade, ativação/reset por token e telas multiusuário continuam nas ISSUE-160–163. Nesta issue, a revogação de sessões comuns na troca de senha é exercitada pelo `UpdateUser` existente; não foi aberto endpoint de gestão ao usuário comum. O login HTML conserva seu redirecionamento existente; o destino administrativo responde 403 para o usuário comum, com apenas mensagem e formulário de logout protegido (correção posterior ao Reviewer).

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


## Correções dos achados 1, 2 e 3 do Reviewer — 18/11/2026

Parecer original preservado em `PARECER-REVIEWER.md`, com seu comentário original na tarefa, ambos incluídos no commit de correção. Execução direta pelo Codex, sem re-delegação. O parecer analisa `35e5180`; este registro descreve o diff posterior solicitado pelo usuário e não altera o veredito original.

- **Achado 1:** `/admin` resolve a sessão e exige admin antes de carregar a página administrativa. Para uma conta comum, retorna 403 com mensagem traduzida em en/pt-BR e um formulário de logout com CSRF e ticket próprios. A view não recebe User, Profile, Profiles, Keys ou configuração; a navegação administrativa permanece ausente. Somente `POST /admin/logout` passa a usar SessionIdentity; os demais POSTs continuam passando por AdminIdentity. O parsing de formulário, SameOrigin, CSRF e ticket existentes são reutilizados sem exceção. Logout revoga só a sessão atual, limpa o cookie e redireciona para `/admin`, onde o formulário de login volta a aparecer. Não foi implementado fluxo do menu (+), gestão própria ou navegação por permissões.
- **Achado 2:** hash fictício bcrypt gerado uma única vez no pacote com DefaultCost. Login compara a senha com esse hash quando a conta não existe ou está pending; o resultado nunca autentica esses estados. `loginWithCompare` recebe a função de comparação por parâmetro interno, evitando hooks globais mutáveis. `TestRejectedLoginRunsBcrypt159` observa exatamente uma chamada para cada ramo, confere o hash/custo e executa a comparação real; uma função que retorna sucesso também não libera login. Sem medições de tempo. Entrada vazia e falhas de armazenamento conservam seus contratos existentes. O log de falha de login passa a "Login failed" e o 401 genérico a "authenticated session required", com traduções en/pt-BR; credenciais inválidas continuam com a mesma mensagem uniforme.
- **Achado 3:** `TestUserHTMLLoginAndLogout159` percorre formulário/preauth → login user → 403 em en/pt-BR sem dados administrativos → logout → formulário de login. Rejeita CSRF ausente, CSRF de outra sessão, origem externa, GET e replay após revogação; preserva a sessão do administrador e também testa seu logout. `TestUserManagementRoutesStayForbidden159` cobre GETs de track/artist/album e revisão e POSTs de metadata/confirm, negando user e mantendo GETs admin. `TestAdminSessionRejectsOrdinaryAccount159` exercita adminSession e o HTML público real: user sem links de gestão, admin com links.

### Referência de UI

O primeiro make check desta correção passou os testes Go e parou no comparador da referência por diferença apenas no atributo `data-ui-generation`, que inclui o hash de todos os templates. Um verificador local comparou todos os arquivos exportados e exigiu igualdade byte a byte após substituir exclusivamente esse atributo. Somente os 12 atributos das fixtures públicas da ISSUE-073 foram atualizados. Todos os demais bytes de HTML e todos os assets permaneceram idênticos; não houve alteração ou aceite novo de layout. A página 403 nova foi renderizada pelos testes HTTP em ambos os idiomas, com ausência dos controles administrativos verificada.

### Revalidação das correções

- `make check`: passou, incluindo vet/test de todos os pacotes, UI/JavaScript/estilo, comparação de referência e 13/13 jornadas e2e.
- `CGO_ENABLED=1 TUNETRAIL_WEB_DIR="$PWD/assets" go test -race ./internal/admin ./internal/adminweb ./internal/adminui ./internal/webui ./engine/handlers ./engine`: passou nos seis pacotes tocados nesta correção. SQLite não foi alterado nesta rodada; sua corrida e a corrida de senha permanecem cobertas pela validação original e pelo Reviewer.
- Graphify AST atualizado e consultado para loginWithCompare, logoutPage, dispatch e adminSession. Revisão própria do diff incremental conforme Go/deslop: a única exceção de papel é logout, com sessão ativa, SameOrigin e tokens existentes; sem novas rotas de gestão, limites ou alteração de defaults.
- `git diff --check` limpo. ACs 1–3 continuam comprovados. Status Em andamento; sem push, merge, publish ou Done.
