---
id: ISSUE-132
title: '132 — Contrato de identidade: gravação, ocorrência e evento'
status: Aberta
assignee: []
created_date: '2026-11-15T16:43:00Z'
updated_date: '2026-11-15T16:44:00Z'
labels:
  - catalogo
  - banco
milestone: m-11
dependencies:
  - ISSUE-131
documentation:
  - doc-036
type: spike
ordinal: 145000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Recorte 2 do estudo da ISSUE-116 (doc-036). Definir o contrato de identidade e de estatística (gravação, ocorrência num lançamento, evento), alinhado às ISSUE-079, 080, 081 e 092, antes de qualquer migração. Inclui a decisão do usuário: ranking por gravação como padrão, com opção por lançamento.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Fixtures distinguem gravação, ocorrência e evento
- [ ] #2 Decididos: álbum ausente, colisão no mesmo segundo e revisão
- [ ] #3 Exemplo A/B: 2 escutas na gravação e 1 em cada álbum, sem vazamento entre dois perfis
- [ ] #4 Decisão registrada em backlog/decisions após aceite
<!-- AC:END -->
