---
id: ISSUE-147
title: 147 — Scrobbler Android nativo para o TuneTrail
status: Aberta
assignee: []
created_date: '2026-11-16T01:05:00Z'
updated_date: '2026-11-22T20:40:00Z'
labels:
  - ingestao
milestone: m-12
dependencies:
  - ISSUE-145
documentation:
  - doc-038
  - doc-045
priority: low
type: feature
ordinal: 160000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: estudo doc-038. Futuro: app Android que observa a reprodução de outros aplicativos (sessões de mídia e notificações) e envia as escutas ao servidor pela API compatível com ListenBrainz, com a chave do perfil. No iOS isso não é possível (um app só registra o que ele mesmo toca), então o iOS continua dependendo de apps externos que enviem pelo ListenBrainz. Antes de implementar, comparar com scrobblers Android existentes que já aceitam ListenBrainz com URL própria, para não duplicar. Alinhar com a deduplicação e a retenção das ISSUE-079 a 081.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Comparação com scrobblers existentes que aceitam ListenBrainz com URL própria e decisão de construir ou recomendar
- [ ] #2 Escutas de outros aplicativos são capturadas e enviadas ao perfil correto, sem duplicar
- [ ] #3 Falha de rede não perde escutas (fila local) e a deduplicação da ISSUE-081 é respeitada
- [ ] #4 Permissões do Android explicadas ao usuário e testadas
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: Codex
created: 2026-11-22T20:40:00Z

---

O estudo do Pano descreve captura de mídia, pré-processamento e fila offline. Usar os resultados da ISSUE-191 na avaliação de construir um cliente Android próprio ou recomendar a integração existente.

---
<!-- COMMENTS:END -->
