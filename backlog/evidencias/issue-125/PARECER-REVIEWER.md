# Parecer do Reviewer — ISSUE-125 (commit 8a433dc)

**APROVADO.** Nenhum [BUG] executado.

Testes rodados por mim (`-race -count=1`, assets/langs configurados): adminui, adminweb, webassets, webui, admin (ok) e engine `-run 'Account|Navigation|Auth|SignOut|Sidebar|Authorization|Login|Admin'` (ok, 137 s).

## [OK]
- Fluxo de credenciais preservado: a troca de username/senha continua em `admin.ChangeAccount` e a revogação de sessões e limpeza de cookie/preauth ficaram iguais (`internal/adminui/account.go`, `actions.go`); só a apresentação por cartão mudou. CSRF, ticket de envio e SameOrigin seguem no despachante, antes de `accountAction`.
- `/account` exige sessão e `ManageAccount` (qualquer papel); sem sessão mostra o login; `/admin?section=account` redireciona (303) sem quebrar links; `SafeReturn` e `canReturnToRead` aceitam `/account` sem query.
- Acessibilidade: erros com `role=alert`, `aria-invalid`, `aria-describedby` ligando campo, ajuda, aviso e erro; aviso de encerrar todas as sessões ANTES de enviar; foco visível; Sair… secundário; funciona sem JS (fragmentos só com `data-enhance`).
- CSS 44 linhas, só min-width 40/64rem, tokens existentes.

## [RISCO] baixo
- `internal/admin/profiles.go` e `internal/webui/routing.go`: `account` virou slug reservado. Um perfil já existente com esse slug ficaria inacessível. No banco de desenvolvimento não há (slugs verificados) e produção vem da v2 sem perfis; ainda assim vale uma checagem no cutover.
- `accountPage` não restringe o método: `POST /account` renderiza a página (sem mutação, sem risco de CSRF), só destoa de GET/HEAD.
- Após trocar a senha o usuário cai no login sem a mensagem "senha alterada" (melhorar na ISSUE-128).
- `/account?qualquer-coisa` responde 400 (links com parâmetros de rastreio quebram).
- Nome duplicado: só `db.ErrConflict` vira mensagem no cartão; outro erro cai na página genérica, como antes.
- O agente sugeriu a porta 4110 para testar: é a do desenvolvimento do usuário. Use 4125.
