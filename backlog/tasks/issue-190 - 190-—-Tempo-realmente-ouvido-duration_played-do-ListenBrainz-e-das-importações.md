---
id: ISSUE-190
title: >-
  190 — Tempo realmente ouvido: duration_played do ListenBrainz e das
  importações
status: Em andamento
assignee:
  - '@ebrgr'
created_date: '2026-11-22T20:05:00Z'
updated_date: '2026-11-23T19:11:00Z'
labels:
  - ingestao
  - catalogo
milestone: m-11
dependencies:
  - ISSUE-172
references:
  - backlog/evidencias/issue-190/MEDICAO.md
type: feature
ordinal: 203000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: PR gabehf/Koito#379 (fecha o #354; aberto, não mesclado), revisado em 22/11/2026. Hoje o tempo ouvido usa a duração da faixa, mesmo quando só metade foi tocada. Aceitar duration_played (segundos) e duration_played_ms (convertido) do additional_info do ListenBrainz, guardar por escuta e usar COALESCE(duration_played, duração da faixa) nas contagens de tempo ouvido (faixa, álbum, artista, período, Rewind). Casos de teste do PR a reaproveitar: 120 s informado, 90,5 s via _ms convertido, ausente usa a duração da faixa (soma 120+90+200). No TuneTrail: campo no Snapshot (ingest), coluna por escuta com NULL para desconhecido (o PR usa 0, evitar), limite para não passar da duração da faixa, inclusão na exportação/reimportação (ISSUE-183), no relay e na importação do ListenBrainz e do Spotify (ms_played). Não copiar o código do PR: depende do banco e das consultas antigas. Toca o Snapshot e as consultas de tempo ouvido: fazer DEPOIS da ISSUE-172 fase 2, para não conflitar com a remoção do escritor legado. Advisor opcional; re-medir as consultas de tempo ouvido.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Snapshot e migração aditiva 023 guardam segundos inteiros por escuta; desconhecido fica NULL, com limite pela faixa e teste de backup e restauração.
- [x] #2 ListenBrainz aceita 120 s, converte 90,5 s para 90 s e usa 200 s da faixa na ausência, totalizando 410 s.
- [x] #3 Importações ListenBrainz, Spotify e TuneTrail, exportação e relay preservam o tempo ouvido.
- [x] #4 Faixa, álbum, artista, período e Rewind usam o tempo ouvido; planos e tempos antes e depois constam em MEDICAO.md.
- [x] #5 Guia de ingestão documentado e make check aprovado.
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-23T00:38:00Z

---

Decisão do usuário (22/11/2026): esta issue está no mínimo que altera o banco antes do primeiro upgrade de produção e antecede a consolidação das migrações (ISSUE-195). Migração aditiva agora; depois dobrada no arquivo único da 195.

---

author: ebrgr
created: 2026-11-23T19:01:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-190` na branch `feat/issue-190-tempo-realmente-ouvido-duration-played` criada a partir de `main`.

---

author: codex
created: 2026-11-23T19:11:00Z

---

Verificação da implementação: migração aditiva 22→23 com backup restaurável, proteção de downgrade e CHECK. O caso ListenBrainz soma 410 s (120 + 90 + 200), preserva zero explícito e aplica o limite da faixa. Cobertos ingestão e revisão, arquivos ListenBrainz e Spotify, exportação e reimportação TuneTrail, relay, faixa, álbum, artista, período e Rewind. MEDICAO.md registra planos e tempos com 300.080 escutas. make check aprovado (13/13 jornadas E2E) e comparação da referência de UI aprovada (190 arquivos).

---
<!-- COMMENTS:END -->
