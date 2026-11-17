---
id: ISSUE-162
title: 162 — Gestão root de contas e perfis com ativação e recuperação por token
status: Aberta
assignee: []
created_date: '2026-11-17T17:33:00Z'
updated_date: '2026-11-17T17:33:00Z'
labels:
  - admin
  - banco
  - api
milestone: m-10
dependencies:
  - ISSUE-158
  - ISSUE-159
  - ISSUE-160
references:
  - backlog/decisions/decision-010
documentation:
  - doc-041
priority: high
type: feature
ordinal: 175000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: ISSUE-100 (aceita em 2026-11-17). Contrato normativo: doc-041 e decision-010. Padrão de senhas atual mantido (bcrypt e validação existentes), sem novos limites ou bloqueios por usuário ou origem; o limitador de login compartilhado existente fica como está; perfis existentes e novos nascem públicos (require_login=0); o relay segue ativo mesmo com escutas protegidas (consequência aceita). Escopo: somente o root cadastra e exclui contas e perfis (operação conjunta, atômica e confirmada, que remove sessões, tokens, chaves e relay do perfil sem tocar nos outros); qualquer perfil pode ser promovido a admin; ativação sem senha compartilhada e recuperação sem e-mail por token temporário emitido pelo root (uso único, expira, reset revoga sessões); o último admin não pode ser excluído nem rebaixado.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Cadastro e exclusão só pelo root e atômicos; promoção a admin por outro admin; último admin protegido
- [ ] #2 Token temporário de uso único, com expiração; reset revoga sessões; testes de token expirado e reutilizado
- [ ] #3 O root nunca vê senha nem hash em tela, API ou log
<!-- AC:END -->
