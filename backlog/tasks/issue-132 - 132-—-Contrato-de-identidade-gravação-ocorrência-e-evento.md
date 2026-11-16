---
id: ISSUE-132
title: '132 — Contrato de identidade: gravação, ocorrência e evento'
status: Aberta
assignee: []
created_date: '2026-11-15T16:43:00Z'
updated_date: '2026-11-16T16:26:00Z'
labels:
  - catalogo
  - banco
milestone: m-11
dependencies:
  - ISSUE-131
references:
  - backlog/evidencias/issue-131/verificacao.md
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
- [ ] #5 Contrato define preservação durável e revisão/reprocessamento de eventos com alias ambíguo ou MBIDs conflitantes, sem associação silenciosa, com fixtures e isolamento por perfil
- [ ] #6 Contrato define a resposta da API ListenBrainz para conflitos e a continuidade/retomada da importação Last.fm, incluindo prevenção de perda e duplicação de escutas
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: codex
created: 2026-11-16T16:26:00Z

---

Pendências encaminhadas pelo usuário após o aceite da ISSUE-131 em 16/11/2026: atualmente um alias ambíguo ou MBIDs incompatíveis recusam a associação e a escuta não é salva; a importação Last.fm é interrompida e o ListenBrainz recebe HTTP 500 pelo tratamento atual. Esta spike deve definir o contrato de preservação durável do evento e metadados originais, motivo da pendência, revisão e reprocessamento idempotente por perfil; definir também a resposta da API e a continuidade/retomada da importação. Registrar fixtures para esses casos e distribuir a implementação nas ISSUE-133/134 ou tarefas próprias conforme o contrato aprovado, antes de qualquer migração. A correção da ISSUE-131 permanece sem migração e sem fila de pendências.

---
<!-- COMMENTS:END -->
