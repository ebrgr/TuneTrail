---
id: ISSUE-159
title: 159 — Autenticação e sessões para todos os usuários
status: Aberta
assignee: []
created_date: '2026-11-17T17:33:00Z'
updated_date: '2026-11-17T17:33:00Z'
labels:
  - admin
  - api
milestone: m-10
dependencies:
  - ISSUE-158
references:
  - backlog/decisions/decision-010
documentation:
  - doc-041
priority: high
type: feature
ordinal: 172000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: ISSUE-100 (aceita em 2026-11-17). Contrato normativo: doc-041 e decision-010. Padrão de senhas atual mantido (bcrypt e validação existentes), sem novos limites ou bloqueios por usuário ou origem; o limitador de login compartilhado existente fica como está; perfis existentes e novos nascem públicos (require_login=0); o relay segue ativo mesmo com escutas protegidas (consequência aceita). Escopo: o login passa a valer para usuários user e admin (hoje só admin autentica), separando autenticar de autorizar; separar sessão de papel; CSRF e tickets adaptados; troca de senha revoga as sessões e a corrida com login de senha antiga é coberta; sem novos limites ou bloqueios.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Login user e admin com bcrypt e validação atuais; sessão por usuário, isolada
- [ ] #2 CSRF e tickets adaptados ao usuário comum; troca de senha revoga sessões; corrida com senha antiga coberta
- [ ] #3 Nenhum limite novo por usuário ou origem; limitador compartilhado existente intacto (teste)
<!-- AC:END -->
