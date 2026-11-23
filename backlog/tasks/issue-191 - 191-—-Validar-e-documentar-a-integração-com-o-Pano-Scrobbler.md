---
id: ISSUE-191
title: 191 — Validar e documentar a integração com o Pano Scrobbler
status: Aberta
assignee: []
created_date: '2026-11-22T20:40:00Z'
labels:
  - api
  - ingestao
  - docs
dependencies: []
documentation:
  - doc-045
priority: medium
type: spike
ordinal: 204000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Verificar a integração ListenBrainz com servidor personalizado. Usar a raiz /apis/listenbrainz/ e uma chave de teste do perfil, pois o Pano acrescenta 1/. O cliente pede return_msid=true em playing_now e o TuneTrail responde com status ok. Avaliar o resultado no cliente antes de decidir se é preciso alterar o contrato.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Registrar versões e resultados de login com chave válida e revogada, playing_now e single em Android e em uma plataforma desktop.
- [ ] #2 Testar reconexão e import em lote, preservação do horário original, ausência de duplicatas após reenvio e isolamento entre perfis.
- [ ] #3 Publicar a configuração validada no guia do repositório e registrar limitações de compatibilidade. Se houver falha, anexar evidência sem credenciais e propor correção separada quando necessário.
<!-- AC:END -->
