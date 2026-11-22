# Parecer do Reviewer — ISSUE-126 (commits 021f481, e8c74ab, f94fc9e, 9dc0d75)

**APROVADO.** Nenhum [BUG] executado. Fronteira dono/root verificada na prática.

## Verificação própria
- `-race -count=1` em adminui, adminweb, webassets, webui, admin e engine (`Account|Navigation|Auth|SignOut|Sidebar|Authorization|Login|Admin|Redeem|Icon|Profile|Visibility|Import|I18n|Languages`): ok. `make check` do agente: 13/13 E2E.
- Sonda HTTP num binário real (porta 4128, banco novo, três papéis, 26 verificações, todas OK):
  - usuário comum: `/account` e as quatro abas respondem 200 sem nenhum controle de root (`account-token`, `account-role`, criar conta, excluir); a faixa de abas não tem Saúde; `/admin?profile=<outro>`, `/admin?section=profiles`, `/admin?section=keys&profile=<outro>` e `/admin/settings` dão 403; `/account/profile?profile=<outro>` e `?x=1` dão 400; `/account/outra` dá 404; `/admin?profile=<próprio>` redireciona 303 para `/account/profile`; POST em perfil alheio com CSRF próprio dá 403; POST no próprio perfil volta para `/account/profile`.
  - root: lista de contas em `/admin?section=profiles`; contexto de outra conta com ferramentas root na aba Conta; no próprio `/account` não há ferramentas root.
  - visitante: login nas rotas novas.

## [OK]
- Alvo próprio derivado da sessão (`ownProfile`) e autorizado com a mesma política das ações (`AuthorizeProfile`) antes de carregar dados; `renderProfileTab` recusa contexto (`!self`) sem `ManageAccounts`; o servidor decide `self` por `profile.UserID == identity.User.ID`, não pelo formulário.
- Rotas novas entram em `IsPage` e no retorno seguro (`IsAccountPath`, sem query); contratos de POST, CSRF, ticket de uso único e SameOrigin inalterados; erro de ação mostrado no cartão com o mesmo status.
- Saúde fora das abas e no menu (+) como decidido.

## [RISCO] baixo
- Dois arquivos de trava de teste (`engine/eMcLqglk/.tunetrail-*.lock`) tinham sido commitados por engano; removidos num commit.
- Aceite visual (AC4): o CSS das abas, do banner e da lista só passou no verificador de estilo; falta o usuário ver no navegador (Yuu e Pearl, claro/escuro, celular e desktop).
- Com JavaScript, uma ação troca só o cartão da ação: dado alterado que aparece em outro cartão da mesma aba fica velho até recarregar (só o slug tem tratamento).
- `/admin?section=account&profile=<outro>` para root agora abre a aba Conta em contexto, em vez de redirecionar para `/account` (mudança intencional; dono e comum seguem com 403).
- `/account/<aba>` devolve 404 se a sessão não tiver perfil (não ocorre no schema 1:1).
- A lista root consulta `ProfileAccount` por perfil (N consultas): aceitável para poucas contas, observar se a lista crescer.
