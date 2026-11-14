---
id: ISSUE-115
title: >-
  115 — Correções herdadas do Koito: IPv6, listened_at decimal, ping Subsonic e
  importação entre volumes
status: Done
assignee:
  - '@ebrgr'
created_date: '2026-11-14T22:54:00Z'
updated_date: '2026-11-14T23:44:00Z'
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
- [x] #1 Endereço IPv6 aceito no bind
- [x] #2 listened_at decimal aceito no envio e na importação
- [x] #3 Ping Subsonic com versão semântica
- [x] #4 Importação funciona com pastas em volumes diferentes
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: ebrgr
created: 2026-11-14T23:37:00Z

---

Início: branch `fix/issue-115-tarefa` criada a partir de `main`.

---

author: ebrgr
created: 2026-11-14T23:41:00Z

---

Verificação: make check (12/12 jornadas, referência sem diferenças) e make api.race aprovados; testes de regressão para IPv6 (6 endereços), listened_at decimal (unidade e importação ListenBrainz), ping Subsonic v=1.13.0 e movimentação entre volumes simulando EXDEV, inclusive recusa de symlink.

---

author: ebrgr
created: 2026-11-14T23:43:00Z

---

Aceite: aprovada pelo usuário em 14/11/2026. Publicação local autorizada na mensagem: "aceito, publique". Push não solicitado.

---

author: ebrgr
created: 2026-11-14T23:44:00Z

---

Publicação: versão 0.14 (tag `v0.14`), branch `fix/issue-115-tarefa` integrada ao `main` por fast-forward.

---
<!-- COMMENTS:END -->
