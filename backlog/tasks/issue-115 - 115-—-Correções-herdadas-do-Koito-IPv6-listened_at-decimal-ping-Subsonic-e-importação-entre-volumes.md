---
id: ISSUE-115
title: >-
  115 — Correções herdadas do Koito: IPv6, listened_at decimal, ping Subsonic e
  importação entre volumes
status: Aberta
assignee: []
created_date: '2026-11-14T22:54:00Z'
updated_date: '2026-11-14T22:54:00Z'
labels:
  - ingestao
milestone: m-11
dependencies: []
type: bug
ordinal: 124000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: estudo ISSUE-114 (doc-034). Bugs do Koito confirmados no código do TuneTrail:
- Koito #74: cfg.ListenAddr usa fmt.Sprintf("%s:%d") e quebra com endereço IPv6; usar net.JoinHostPort.
- Koito PR #373: listenbrainz.Listen.ListenedAt é int64; clientes que enviam listened_at decimal falham no envio e na importação ListenBrainz.
- Koito PR #349 / #211: ping Subsonic envia v=1; LMS exige versão semântica (1.13.0).
- Koito #185: importer.go e profile_import.go usam os.Rename; falha quando import/ e import_complete/ estão em volumes diferentes (EXDEV). Copiar e remover como alternativa.
Cada correção com teste de regressão.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Endereço IPv6 aceito no bind
- [ ] #2 listened_at decimal aceito no envio e na importação
- [ ] #3 Ping Subsonic com versão semântica
- [ ] #4 Importação funciona com pastas em volumes diferentes
<!-- AC:END -->
