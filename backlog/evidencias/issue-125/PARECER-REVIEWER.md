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

---

# Reauditoria (commits 84a5899, 5baff68: resgate de token, cores e estados)

**APROVADO.** Nenhum [BUG] executado. Testes: adminui, adminweb, webassets, webui, admin e engine `-run '…|Redeem'` passam com `-race`.

## [OK]
- Falha de CSRF/preauth em `/admin/redeem` e `/admin/login` agora volta ao formulário com token novo e a mensagem "Esta página expirou…" (403), em vez de "sem permissão". O username digitado é mantido; token de conta e senha nunca são ecoados. Nenhuma checagem de CSRF/SameOrigin foi afrouxada.
- Resgate: confirmação de senha com erro por campo, regras visíveis ligadas por `aria-describedby`, `autocomplete=new-password`, sucesso redireciona para o login com o aviso "Senha definida…"; sessão já aberta que cair em `/admin?redeemed` vai para `/account`.
- `check_web_style.py` passa a barrar cor fixa e redefinição de foco em `account.css` (com testes do verificador no `make check`).

## [RISCO] baixo
- `peekRedeemUsername` relê o início do corpo (até 8 KiB) antes da checagem de CSRF para preservar o username: funciona e o valor é escapado pelo template, mas é um acoplamento incomum com o corpo da requisição.
- Hover/pressionado do botão primário (`.account-action`) usa `color-mix` do primário com a cor do texto: o botão primário da administração não tinha hover definido, então é uma derivação, não um padrão já existente. Vale conferir no navegador (Yuu e Pearl).
- O foco nas telas de administração é o anel global de `admin.css` (2px primário, offset 4px), não o de 1px do app público: coerente com a administração, diferente da referência que o usuário citou (app público).
- `msgPasswordInvalid` da `/account` diz "8 e 72 caracteres", mas a regra real é 8 caracteres e 72 BYTES (`validAccountPassword`); o resgate já diz "72 bytes".
- `messages.go` teve a tabela realinhada pelo gofmt (diff grande, sem mudança de conteúdo).
- A checagem de estilo cobre só `account.css`; o bloco de resgate usa as mesmas classes em `admin.html`, verificadas apenas pelo CSS.
