---
id: ISSUE-192
title: 192 — Exibir e filtrar a origem das escutas na central de saúde
status: Aberta
assignee: []
created_date: '2026-11-22T20:40:00Z'
labels:
  - admin
  - ingestao
dependencies:
  - ISSUE-184
documentation:
  - doc-045
priority: medium
type: enhancement
ordinal: 205000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Avaliar a disponibilidade de submission_client, submission_client_version e media_player nos recibos da Inbox. Identificar de onde vieram escutas com falhas ou duplicatas, mantendo a atribuição por chave e por importação.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Documentar quais campos existem nos recibos novos e históricos, incluindo variantes do Pano e importações.
- [ ] #2 Exibir cliente, versão e player quando disponíveis, indicar origem ausente e permitir filtrar por cliente e player.
- [ ] #3 Testar recibos legados, origem ausente, metadados declarados pelo cliente e isolamento entre perfis, sem exibir segredos.
<!-- AC:END -->
