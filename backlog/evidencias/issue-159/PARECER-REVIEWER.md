# ISSUE-159 — Parecer do Reviewer

Alvo: branch `feat/issue-159-autenticacao-e-sessoes-para-todos`, commit `35e5180` sobre `main` (v0.36). Revisão adversarial de segurança feita diretamente pelo Reviewer (Claude), sem re-delegação. Contexto: tarefa (3 ACs), doc-041 §§ sessões/formulários, decision-010 item 5, `VERIFICACAO.md`, diff `main...HEAD` e grafo (graphify) dos consumidores de `Authenticate`/`SessionIdentity`/`GetUserBySession`.

## Achados

- [RISCO] internal/adminui/admin.go:231-240 + internal/adminui/actions.go:57 — o login HTML (`POST /admin/login`) agora aceita `role=user` e redireciona para `/admin`, que responde 403 sem formulário. `POST /admin/logout` passa por `AdminIdentity` e também nega o usuário comum. Resultado: quem entra pelo formulário HTML fica preso numa página 403 sem logout HTML até a sessão expirar (24h ou 30 dias). Enquanto o cookie existir, o navegador também não mostra o formulário de login em `/admin` a um administrador, porque `ErrForbidden` não cai no ramo `loginPage`. O logout JSON funciona. Sem escalonamento de privilégio, mas é um beco sem saída novo, alcançável só a partir da 159. Correção mínima: aceitar `/admin/logout` com `SessionIdentity`+CSRF da sessão e mostrar o formulário de logout na página 403. Outra opção é registrar explicitamente na ISSUE-163/157 (doc-041 já prevê "logout POST protegido também para usuário comum") como gate de aceite antes do corte multiusuário.
- [RISCO] internal/admin/auth.go:73 — o login retorna antes do bcrypt quando a conta não existe ou está `pending`. Antes, o oráculo de tempo distinguia apenas admins ativos; agora distingue qualquer conta ativa (rápido = inexistente/pending, lento = ativa). As mensagens são uniformes (401 "invalid credentials" / "Invalid username or password."). O impacto é baixo, porque os logins derivam de slugs públicos (ISSUE-158). Correção barata, sem novo controle: rodar `bcrypt.CompareHashAndPassword` contra um hash fixo de custo `DefaultCost` nesse ramo. Pode ficar para a 162/164 se o usuário aceitar.
- [RISCO] engine/auth_issue159_test.go:24 — o inventário testado de rotas cobre `/admin`, settings, delete-review, export, `profile/1/save`, `/apis/web/v1/user` e `management`. Não cobre `GET/POST /{slug}/manage/...` com sessão de user, `webui.adminSession` com user (nav administrativa pública) nem `POST /admin/logout`. O código está correto (todos passam por `AdminIdentity`/`RequireAdmin`), mas faltam testes de regressão para a 160. Sugestão: incluir essas rotas na tabela do teste.
- [RISCO] engine/handlers/auth.go:30 / internal/adminweb/errors.go:56 — o log "Administrative login failed" e a mensagem pública 401 "administrator session required" ficam imprecisos para contas comuns. É cosmético; ajustar junto com a 163.

## Conferido

- [OK] Escalonamento: os únicos consumidores de sessão são `RequireAdmin` (via `validateSession`), que mantém `u.Role != admin → 403` (engine/middleware/profiles.go:99), e os adaptadores HTML. Todas as entradas HTML (`dispatch`: settings, `/admin`, export, todo POST/`action`; `reviewProfile`; `media`/catalog review/profile delete) usam `AdminIdentity`, e `webui.adminSession` também. Todas as rotas mutáveis/privadas de `/apis/web/v1` estão sob `RequireAdmin`; `ChangeAccount` e os serviços admin continuam chamando `Authorize`.
- [OK] Papel relido do banco a cada requisição (`GetUserBySession`); rebaixamento retira autorização sem invalidar a autenticação (testado).
- [OK] CSRF ligado ao UUID da sessão (HMAC, TTL, comparação constante); tickets ligados à sessão, uso único, 30 min, limite de 4096; rejeição entre contas e replay testados com dois usuários comuns reais.
- [OK] `SaveAuthenticatedSession` mantém `id + password hash + status='active'` no INSERT…SELECT; só caiu `role='admin'`. A corrida login×troca de senha é coberta para user (`pausedLogin`) e para admin (testes existentes); `UpdateUser` revoga todas as sessões da conta na mesma transação; a sessão de outra conta é preservada.
- [OK] Conta `pending`: recusada em `Login`, `Authenticate`, `GetUserBySession` e `SaveAuthenticatedSession`. A ingestão por API key de perfil pending segue o contrato (doc-041) e não confere sessão.
- [OK] Cookie `tunetrail_session` inalterado: HttpOnly, SameSite=Strict, Secure por TLS/X-Forwarded-Proto, expiração do store. Logout JSON revoga só a sessão atual.
- [OK] Limitador: a mesma instância global `httprate.Limit(10, 1m)` é compartilhada entre HTML e JSON, e `engine/routes.go` não mudou; o teste confirma que outro nome/endereço não abre orçamento novo. Nenhum limite novo por usuário/IP.
- [OK] Regressões: admin padrão, relay, perfis legados e API ListenBrainz por token não foram tocados.
- [OK] AGENTS.md: comentários em inglês nas funções alteradas; CHANGELOG em `[Unreleased]`; commit prefixado `ISSUE-159:`.
- [OK] Testes: `CGO_ENABLED=1 TUNETRAIL_WEB_DIR=$PWD/assets go test -race -count=1` em admin, adminweb, adminui, db/sqlite, webui e engine passou (rodado pelo Reviewer).

## Veredito

**APROVADO COM RESSALVAS.** Nenhum caminho dá a um usuário comum acesso administrativo. Resta decidir o beco sem saída do login/logout HTML para usuário comum: corrigir agora ou registrar como gate da 163/157. O Advisor é recomendado (autenticação é área de recomendação; @Opus `effort=high` ou @Sol `effort=medium`), com foco no oráculo de tempo e no fluxo HTML user. Decisão do usuário.
