---
id: ISSUE-160
title: 160 — Autorização de gestão própria e administração global
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
  - ISSUE-159
references:
  - backlog/decisions/decision-010
documentation:
  - doc-041
priority: high
type: feature
ordinal: 173000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: ISSUE-100 (aceita em 2026-11-17). Contrato normativo: doc-041 e decision-010. Padrão de senhas atual mantido (bcrypt e validação existentes), sem novos limites ou bloqueios por usuário ou origem; o limitador de login compartilhado existente fica como está; perfis existentes e novos nascem públicos (require_login=0); o relay segue ativo mesmo com escutas protegidas (consequência aceita). Escopo: o dono maneja somente os seus dados, chaves, importação e exportação; root maneja todos os perfis e o servidor; catálogo compartilhado (capas, aliases, merges) só root; papel lido no servidor a cada requisição; último admin ativo protegido em transação; sem mass assignment de role, status ou dono.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Matriz de permissões do doc-041 coberta por testes (visitante, usuário, root) em HTML, JSON e serviços
- [ ] #2 Último admin protegido em transação, com teste concorrente; usuário não se promove
- [ ] #3 Catálogo compartilhado exclusivo do root; chave de ingestão não administra nem lê dados privados
<!-- AC:END -->
