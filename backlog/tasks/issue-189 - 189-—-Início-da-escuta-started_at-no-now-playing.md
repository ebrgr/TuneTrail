---
id: ISSUE-189
title: 189 — Início da escuta (started_at) no now-playing
status: Aberta
assignee: []
created_date: '2026-11-22T20:05:00Z'
labels:
  - ingestao
  - ui
milestone: m-13
dependencies: []
type: feature
ordinal: 202000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: PR gabehf/Koito#380 (aberto, não mesclado), revisado em 22/11/2026. O now-playing guarda hoje só o id da faixa em memkv (internal/listening/now_playing.go, chave now-playing:<perfil>). Guardar também o instante de início e expor started_at (Unix, segundos) em /apis/web/v1/now-playing, para clientes externos e para uma barra de progresso na UI. Cuidados: (1) um playing_now do ListenBrainz normalmente NÃO traz listened_at (o PR do Koito só passa porque o teste foi editado para incluí-lo): sem listened_at, usar o momento de recebimento; (2) manter a expiração do cache pela duração da faixa e a limpeza de entrada obsoleta da ReadNowPlaying; (3) isolamento por perfil como hoje; (4) atualizar docs e testes (com e sem listened_at). Não altera a ingestão de escutas definitivas. Pequena; pode ser feita a qualquer momento.
<!-- SECTION:DESCRIPTION:END -->
