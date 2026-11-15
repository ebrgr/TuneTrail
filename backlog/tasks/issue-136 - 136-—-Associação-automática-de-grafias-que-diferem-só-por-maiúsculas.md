---
id: ISSUE-136
title: 136 — Associação automática de grafias que diferem só por maiúsculas
status: Aberta
assignee: []
created_date: '2026-11-15T16:43:00Z'
updated_date: '2026-11-15T16:44:00Z'
labels:
  - ingestao
  - catalogo
milestone: m-11
dependencies:
  - ISSUE-134
documentation:
  - doc-036
type: feature
ordinal: 149000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Recorte 6 do estudo da ISSUE-116 (doc-036), Koito #359. Decisão do usuário: associar automaticamente grafias que diferem só por maiúsculas, quando houver uma única candidata e nenhum conflito de MBID. A associação vale para o catálogo compartilhado, portanto para todos os perfis.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Regra Unicode determinística e versionada; grafia original de cada escuta preservada
- [ ] #2 Associa sozinho só com candidata única e sem conflito de MBID; ambiguidades vão para revisão
- [ ] #3 Cada associação fica registrada com a versão da regra e pode ser desfeita
- [ ] #4 Vale para faixas, artistas e álbuns, com testes de colisão
<!-- AC:END -->
