---
id: ISSUE-133
title: 133 — Migração aditiva para gravações e ocorrências com IDs compatíveis
status: Aberta
assignee: []
created_date: '2026-11-15T16:43:00Z'
updated_date: '2026-11-15T16:44:00Z'
labels:
  - banco
milestone: m-11
dependencies:
  - ISSUE-132
  - ISSUE-080
documentation:
  - doc-036
type: feature
ordinal: 146000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Recorte 3 do estudo da ISSUE-116 (doc-036). Migração aditiva: faixas atuais viram ocorrências com os mesmos IDs, cada uma ganha uma gravação, escutas migram 1:1 com origem legacy. Coordenar a numeração do schema com a ISSUE-080; backup e restauração validados antes.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Banco com vários perfis, aliases, MBIDs nulos e FKs migra com contagem de eventos idêntica por perfil
- [ ] #2 Nenhuma fusão automática por título; mapeamento old_track_id → ocorrência/gravação guardado
- [ ] #3 IDs e URLs anteriores continuam válidos
- [ ] #4 Interrupção simulada reverte a transação; restauração do backup comprovada
<!-- AC:END -->
