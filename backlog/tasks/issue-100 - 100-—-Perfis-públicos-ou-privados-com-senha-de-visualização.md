---
id: ISSUE-100
title: >-
  100 — Aplicativo multiusuário: contas com senha, administrador atribuível e
  visibilidade das escutas por perfil
status: Aberta
assignee: []
created_date: '2026-11-14T00:39:00Z'
updated_date: '2026-11-17T16:39:00Z'
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
<!-- COMMENTS:END -->
