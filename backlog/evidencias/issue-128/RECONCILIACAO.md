# ISSUE-128 — Reconciliação (base v0.65)

Auditoria estática de `main` em 22/11/2026, antes de qualquer código da 128 reiniciada. Compara os cinco itens do reescopo (comentário #4 da tarefa) com o que as ISSUE-125 (v0.63) e 126 (v0.64) já entregaram.

## O que já existe (não refazer)

| Área | Estado | Onde |
| --- | --- | --- |
| Resgate de token | Pronto: regras da senha, confirmação, caixa de orientação (`admin/redeem-notice`), mensagem «página expirou» com formulário novo, `Referrer-Policy: same-origin` | `admin.html` (ramo `.Redeem`), `account_tokens.go`, `ServeHTTP` |
| Destino pós-login | `/account` para qualquer papel; root vai a `/admin`; retorno só por `SafeReturn` + `canReturnToRead` (valida rota, query, perfil e permissão do ator recém-autenticado) | `login_return.go`, `adminweb/return.go` |
| Conta e abas | `/account` e `/account/{profile,keys,history,relay}`, shell `admin/account-shell`, modo contexto do root, lista de contas | `account-tabs.html`, `account.html`, `accounts-list.html`, `account_tabs.go`, `account.css` |
| Mensagens com classes | `.account-message` / `.account-error` / `.account-warning` / `.form-help` com `role=alert` (erro) e `role=status` (aviso) **só dentro dos cartões de conta** | `account.css`, `account.html`, `account-tabs.html` |
| Confirmação de saída | `GET /admin/sign-out` + `POST /admin/logout` (157) | `sign_out.go` |
| Textos do estudo | `admin.message.sign_in_to_continue`, `page_forbidden`, `ui.this_profile_or_page_is_unavailable`, `invalid_username_or_password`, `page_expired` já existem em `en` e `pt-BR` | `langs/*.json`, `messages.go` |

## O que falta (os cinco itens do reescopo)

1. **Login em cartão central + rota neutra `GET /login`.** Hoje o login é um `<form>` solto dentro de `admin.html` (ramo `.Login`), sem cartão, só em `GET /admin` (sem sessão), `/account*`, `/admin/settings` etc. Não existe `/login` (nem como slug reservado: `login` ainda é um slug de perfil válido). O link «Entrar» da navegação aponta para `/admin?next=…` (`webassets.NavAccess.Login`). Faltam: rota, slug reservado, retorno por `/login?next=`, sessão expirada com «Entre para continuar.» (já existe o texto; falta o estado explícito e o link coerente), e a mensagem «Senha alterada» após `account/password` (hoje o 303 vai a `/account`, que cai no login sem aviso).
2. **401/403/404 no shell neutro.** Existe `failForSession` (403 só com `LogoutOnly`) e `fail` → `render` (admin) e `pages.NotFound` (público). O erro do admin usa o título «Status da solicitação» e o mesmo `<main>` de formulários; o 404 do admin não oferece entradas coerentes com a sessão e o `Locale.Return` fixo em `/`. Falta: um bloco de erro único (título, mensagem da tabela do estudo, entradas Minha conta/Diretório/Entrar/Sair…), 403 sem carregar o recurso, 404 idêntico para ausência e permissão (sem retorno que revele a causa).
3. **Componente único de mensagem.** Hoje há ≥ 12 marcações próprias: `<p role="alert">`/`<p role="status">` soltos em `admin.html`, `manage.html`, `settings.html` e as variantes `account-message` repetidas ~10× em `account.html`/`account-tabs.html`. Falta o parcial `partial/message` (`admin/message`) com classe/tokens únicos, `role` por tipo, `id` para `aria-describedby`, e o uso em login, resgate, conta, abas, revisões e erros.
4. **Revisões em cartões.** `catalog-review`, `profile-review` e `profile-delete` são texto corrido (`<p>`, `<h2>`, `<ul>`) sem cartão, sem separação resumo/lista/confirmação e sem zona de perigo; O plano do catálogo é de uso único por token do servidor (`catalogPlans`), e o apagar de escutas pessoal soma o ticket de envio. Escopo atual: o catálogo exige `ManageCatalog` (root) em `PreviewCatalogMutation`/`ApplyCatalogMutation`; o apagar de escutas pessoal exige dono ou root (`authorizePersonalPlan` → `AuthorizeProfile`); `profile-review` exige `ManageAccounts` (root). Fronteira a conferir e cobrir em teste.
5. **Cores e estados.** `account.css` já usa só tokens. Faltam os novos componentes (cartão de login, mensagem, zona de perigo das revisões) seguirem o padrão (`button.large-button`, tokens, hover/active/foco, alvos de 44px) e `make check` (`scripts/check_web_style.py`) verde.

## Decisões de implementação (sem produto em aberto)

- `/login` é a rota canônica; `/admin/login` (POST) e o formulário atual continuam. `GET /admin` sem sessão continua mostrando o mesmo cartão (a ação do formulário é sempre `POST /admin/login`).
- Sessão válida em `GET /login` → 303 para o destino validado (`next` ou fallback do papel), sem renderizar o formulário.
- `login` entra na lista de slugs reservados (`admin/profiles.go`, `webui/routing.go`, `adminweb/return.go`, `adminui/routing.go`). Risco registrado: um perfil já existente com slug `login` deixaria de resolver (a 125 fez o mesmo com `account`).
- Não toca credenciais, bcrypt, sessões nem o lock do resgate: só apresentação, rotas GET e o destino do 303 após a troca de senha.
