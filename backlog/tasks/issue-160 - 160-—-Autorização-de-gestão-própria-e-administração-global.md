---
id: ISSUE-160
title: 160 — Autorização de gestão própria e administração global
status: Em andamento
assignee:
  - '@ebrgr'
created_date: '2026-11-17T17:33:00Z'
updated_date: '2026-11-19T00:07:00Z'
labels:
  - admin
  - api
milestone: m-10
dependencies:
  - ISSUE-158
  - ISSUE-159
references:
  - backlog/decisions/decision-010
  - backlog/evidencias/issue-160/AMBIGUIDADE-IMPORTACAO.md
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

## Comments

<!-- COMMENTS:BEGIN -->
author: ebrgr
created: 2026-11-19T00:05:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-160` na branch `feat/issue-160-autorizacao-de-gestao-propria-e` criada a partir de `main`.

---

author: codex
created: 2026-11-19T00:07:00Z

---

Verificação parcial: análise estática identificou ambiguidade entre a importação própria permitida no doc-041 §2 e a criação de capas/aliases compartilhados pelo ImportKoitoFile (internal/importer/koito.go:105,141,184). Execução interrompida conforme instrução explícita, aguardando decisão registrada em backlog/evidencias/issue-160/AMBIGUIDADE-IMPORTACAO.md. Nenhum runtime alterado, nenhum AC marcado; make check e race não executados. Mantido Em andamento.

---
<!-- COMMENTS:END -->
