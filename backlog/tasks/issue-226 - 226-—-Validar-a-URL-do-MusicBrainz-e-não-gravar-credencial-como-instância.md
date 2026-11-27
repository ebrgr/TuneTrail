---
id: ISSUE-226
title: 226 — Validar a URL do MusicBrainz e não gravar credencial como instância
status: Aberta
assignee: []
created_date: '2026-11-27T15:21:00Z'
updated_date: '2026-11-27T15:21:00Z'
labels:
  - ingestao
  - config
milestone: m-13
dependencies: []
priority: medium
type: bug
ordinal: 239000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Decisão do usuário em 27/11/2026 (C7). Risco 1 do Reviewer da ISSUE-216: MusicBrainz.Instance() devolve a URL crua e NewMusicBrainz não a valida, ao contrário do Subsonic, que rejeita usuário embutido, consulta e fragmento e usa um hash como instância. Uma URL de espelho com credencial em KOITO_MUSICBRAINZ_URL seria gravada em metadata_values.instance, nas linhas do job e nos backups. A exportação já ofusca a instância.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 URL validada (http ou https, sem usuário embutido, consulta nem fragmento), com aviso claro quando inválida
- [ ] #2 Instância sem credencial (hash do endpoint ou esquema e host), com teste de que a credencial não chega ao banco
- [ ] #3 Linhas já gravadas com a URL crua tratadas
<!-- AC:END -->
