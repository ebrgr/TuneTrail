---
id: ISSUE-102
title: 102 — Container sem root com migração de permissões
status: Aberta
assignee: []
created_date: '2026-11-14T00:39:00Z'
labels:
  - operacao
milestone: m-15
dependencies:
  - ISSUE-063
priority: medium
type: chore
ordinal: 102000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: doc-017 (ISSUE-034) e doc-010 §8. `USER 65532`, nota de migração de ownership dos volumes existentes (inclusive Unraid) e teste com volume antigo.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Imagem roda sem root
- [ ] #2 Volumes existentes migrados com instrução testada
- [ ] #3 Documentação de atualização
<!-- AC:END -->
