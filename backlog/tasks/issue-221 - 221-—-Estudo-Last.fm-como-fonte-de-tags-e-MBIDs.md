---
id: ISSUE-221
title: '221 — Estudo: Last.fm como fonte de tags e MBIDs'
status: Aberta
assignee: []
created_date: '2026-11-24T19:53:00Z'
updated_date: '2026-11-24T19:54:00Z'
labels:
  - ingestao
  - docs
milestone: m-13
dependencies: []
references:
  - backlog/evidencias/issue-055/ESTUDO-COMPLEMENTAR.md
documentation:
  - doc-048
priority: low
type: spike
ordinal: 234000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: decisão do usuário em 24/11/2026 (subsistema de enriquecimento compartilhado, começando pelas ISSUE-174 e 175) e o fluxo que ele descreveu: banco já salvo, Navidrome, MusicBrainz, depois ListenBrainz e os serviços comerciais. A documentação de scrobbling do Navidrome informa que, a partir de 1º/12/2026, o titular da chave de API deve ter Last.fm Pro. Avaliar o efeito no cliente de imagens existente (internal/images/lastfm.go) e no provedor. Verificar com chamadas reais, sem credenciais gravadas: o que track.getInfo e track.getTopTags devolvem (toptags; se o mbid vem para a faixa e não só para o álbum), contagem de uso, limites de requisição, chave por usuário e custo, e os termos de armazenamento e uso das tags. Decidir se entra, como fonte de metadados da cadeia B e, se devolver MBID, da cadeia A, sempre desligada por padrão. Se entrar, abrir a issue do provedor.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Contrato verificado com respostas reais sanitizadas: campos de tags e MBID e comportamento sem dado
- [ ] #2 Efeito da exigência de Last.fm Pro de 1º/12/2026 no cliente de imagens e no provedor
- [ ] #3 Termos de armazenamento e uso das tags, limites e custo registrados
- [ ] #4 Decisão adotar ou não adotar, e issue do provedor aberta se adotar
<!-- AC:END -->
