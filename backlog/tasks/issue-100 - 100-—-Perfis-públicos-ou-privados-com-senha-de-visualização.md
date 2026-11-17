---
id: ISSUE-100
title: >-
  100 — Aplicativo multiusuário: contas com senha, administrador atribuível e
  visibilidade das escutas por perfil
status: Em andamento
assignee:
  - '@ebrgr'
created_date: '2026-11-14T00:39:00Z'
updated_date: '2026-11-17T17:17:00Z'
labels:
  - admin
  - ui
milestone: m-14
dependencies:
  - ISSUE-074
priority: high
type: feature
ordinal: 100000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: ROADMAP item 2, doc-010 §7 e pedido do usuário em 2026-11-17. Substitui o escopo anterior ('senha simples por perfil só para visualização; somente a conta de administração apaga dados') por um aplicativo realmente multiusuário. Requisitos do usuário: (1) todo perfil tem login com usuário e senha; (2) existe a opção de administrador, atribuível a qualquer usuário; (3) o usuário com a opção de admin vê e gerencia TODOS os perfis (dados, chaves, cadastro, exclusão); (4) por perfil, uma opção de mostrar ou não as escutas a quem não está logado; (5) o manejo dos dados (editar, excluir, importar, chaves) só é possível com usuário logado, cada usuário nos seus perfis e o admin em todos. Isto revisa o modelo do estudo doc-021/doc-022 (administração única, perfis públicos sem senha, sessões ligadas só à conta administrativa) e a decisão de acesso da ISSUE-123/144; exige decisão de arquitetura (a registrar como decision-010) e a migração dos perfis e da conta administrativa existentes. Revisão obrigatória do Advisor antes de implementar (autenticação, sessões, senha, autorização: AGENTS.md, @Sol effort=medium no Codex). Coordenar com ISSUE-157 (sair), 153, 074 e as páginas de administração (125 a 128). Fora de escopo aqui: contas de convidado, OAuth de terceiros.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Parecer do Advisor registrado e decisão de arquitetura aceita pelo usuário (modelo usuário, perfil e papel de administrador)
- [ ] #2 Login com usuário e senha para todos os perfis; senha com hash adequado, limite de tentativas e sessões isoladas por usuário
- [ ] #3 Administrador atribuível a qualquer usuário; o admin vê e gerencia todos os perfis, e cada usuário comum só os seus (testes de permissão)
- [ ] #4 Opção por perfil de mostrar ou ocultar as escutas a quem não está logado; perfis ocultos não vazam em páginas, API, busca nem widget
- [ ] #5 Manejo de dados somente com usuário logado, com CSRF e autorização por perfil; migração testada a partir do banco atual (conta administrativa e perfis existentes) sem perder escutas
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-17T16:39:00Z

---

Reescopo pedido pelo usuário (2026-11-17): de 'senha de visualização' para aplicativo multiusuário real (contas com usuário e senha, admin atribuível, visibilidade das escutas por perfil, manejo de dados só logado). Estudo do Advisor a seguir, usando doc-021/doc-022, doc-010 §7 e o código atual de contas e sessões.

---

author: ebrgr
created: 2026-11-17T16:39:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-100` na branch `feat/issue-100-aplicativo-multiusuario-contas-com-senha` criada a partir de `main`.

---

author: advisor
created: 2026-11-17T17:01:00Z

---

Parecer do Advisor (Modo B): APROVADO como direção de arquitetura; detalhes pendentes de decisão. Estudo completo [PLANO ISSUE-100] no ai-memory, ID 01a1ea74-9975-7821-8bc6-111687acf270, plans/issue-100-advisor-multiusuario.md. Incorpora confirmações do usuário: cada login corresponde a um único perfil; raiz lista todos com nome/avatar; checkbox Exigir login para ver as escutas restringe leitura ao dono/admin; toda gestão exige sessão; qualquer perfil pode receber Administrador (root), atribuído só por admin, que gerencia todos e o servidor. Plano contém inventário do código, três alternativas de schema, matriz de permissões, proteção de imagens/API/cache, Argon2id e bcrypt legado, sessões/CSRF/tickets, ativação/reset sem e-mail, migração com IDs/chaves preservados coordenada com ISSUE-080/133, telas/testes/riscos/etapas e texto proposed da decision-010. Perguntas restantes: cadastro/exclusão só root; padrão de visibilidade novos/legados; recuperação por token; consentimento de relay protegido; fronteira de edição pessoal versus catálogo global. Nenhum código alterado, teste executado, AC marcado ou status alterado; sem criar issues/decision accepted, commit ou publicação.

---

author: ebrgr
created: 2026-11-17T17:17:00Z

---

Decisões do usuário (2026-11-17) sobre as perguntas do Advisor: 1) Cadastro e exclusão de contas e perfis só pelo admin (root): SIM. 2) Perfis já existentes ficam PÚBLICOS e os novos também nascem públicos (require_login desmarcado por padrão; a proteção é uma escolha posterior). 3) Recuperação sem e-mail por token temporário emitido pelo root: SIM. 4) A proteção das escutas NÃO pausa o relay: o relay continua ativo tanto para o recebimento de escutas quanto para a realização do relay de saída. 5) Capas, aliases e merges compartilhados continuam exclusivos do root, com cada usuário gerenciando as próprias escutas: SIM. Observação do usuário: manter o padrão de senhas atual e não escalonar a segurança (sistema não crítico): sem Argon2id novo, sem mínimo de 15 caracteres, sem bloqueio nem limite novo; tentativas ilimitadas por usuário e sem considerar a origem.

---
<!-- COMMENTS:END -->
