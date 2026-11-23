# Parecer do Reviewer — ISSUE-128 (commits a109905, 6f9151f, e9935b8, 3c2f021)

**APROVADO.** Nenhum [BUG] executado. Credenciais, bcrypt, sessões e lock do resgate não foram tocados (sem Advisor).

## Verificação própria (22/11/2026)
- `-race -count=1` em adminui, adminweb, webassets, webui e admin: ok. `make check` do agente: 13/13 E2E.
- Sonda HTTP no binário real (porta 4129, banco novo, papéis admin, ana, beto e visitante): `/admin` e `/admin/settings` como comum dão 403; perfil inexistente 404; `/account` e abas 200; visitante vê o formulário de login em `/login`, `/account`, `/admin`, `/admin/health`; CSRF inválido no login 403; senha errada e usuário inexistente dão o mesmo 401 uniforme; delete-review do perfil do root: comum 403, visitante 401, root 200.
- Quem já está logado e abre `/login` (com ou sem `next` externo, `//host` inclusive) vai para `/account`: sem redirecionamento aberto.

## [RISCO] baixo
- `/admin?profile=<existente de outro>` dá 403 e `/admin?profile=<inexistente>` dá 404: revela a existência de IDs (sequenciais). Comportamento anterior à 128; vale unificar no 404 em issue futura (ISSUE-196 cobre a auditoria de rotas).
- Slug `login` virou reservado (como `account`): um perfil já existente com esse slug deixaria de resolver (não há no banco de desenvolvimento; produção vem da v2 sem perfis).
- Páginas protegidas sem sessão respondem 200 com o formulário (o estudo da 163 previa 401): mantido para não quebrar contratos.
- Contraste do botão principal no tema Pearl (2,69:1), anterior à 128, não alterado.
