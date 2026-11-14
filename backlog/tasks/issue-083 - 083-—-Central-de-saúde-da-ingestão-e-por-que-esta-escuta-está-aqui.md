---
id: ISSUE-083
title: 083 — Central de saúde da ingestão e "por que esta escuta está aqui"
status: Aberta
assignee: []
created_date: '2026-11-14T00:39:00Z'
updated_date: '2026-11-14T03:48:00Z'
labels:
  - admin
  - ingestao
milestone: m-11
dependencies:
  - ISSUE-082
priority: medium
type: feature
ordinal: 83000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: doc-010 §4.2. `/admin/health` por perfil (último envio, cliente, fila, falhas e motivos, sem tokens) e, na página da escuta, origem, horário recebido, regra aplicada e estado de entrega.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Central de saúde por perfil sem expor segredos
- [ ] #2 Explicação por escuta acessível ao admin
- [ ] #3 Testes de isolamento entre perfis
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-14T03:48:00Z

---

Coordenar com a ISSUE-112 (logs): a central de saúde mostra a contagem de erros e avisos da última hora com link para /admin/logs filtrado.

---
<!-- COMMENTS:END -->
