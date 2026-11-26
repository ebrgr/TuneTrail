---
id: ISSUE-215
title: >-
  215 — Estudo: medir MusicBrainz por nome e ListenBrainz para achar MBIDs
  (regra de aceitação e ordem)
status: Aguardando aceite
assignee:
  - '@ebrgr'
created_date: '2026-11-24T19:53:00Z'
updated_date: '2026-11-26T18:23:00Z'
labels:
  - ingestao
  - validacao
milestone: m-11
dependencies: []
references:
  - backlog/evidencias/issue-055/ESTUDO-COMPLEMENTAR.md
  - backlog/evidencias/issue-215/RELATORIO.md
  - backlog/evidencias/issue-215/medir_issue_215.py
documentation:
  - doc-048
priority: medium
type: spike
ordinal: 228000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: decisão do usuário em 24/11/2026 (subsistema de enriquecimento compartilhado, começando pelas ISSUE-174 e 175) e o fluxo que ele descreveu: banco já salvo, Navidrome, MusicBrainz, depois ListenBrainz e os serviços comerciais. Evidência de 24/11/2026, com chamadas reais e uma faixa pública (sem dados do usuário): (a) o ListenBrainz GET /1/metadata/lookup/ exige token: HTTP 401 sem o cabeçalho Authorization e também com token inválido; (b) o ListenBrainz GET /1/metadata/recording/ por MBID responde 200 sem token; (c) a busca do MusicBrainz funciona sem chave, mas a consulta recording:"Alive" AND artist:"Pearl Jam" devolveu 911 resultados, e os três primeiros têm score 100 com durações de 373, 342 e 331 s: o score não distingue versões, e só duração, créditos e álbum desempatam. Medir: tomar uma amostra de escutas com MBID de referência que NÃO tenha vindo do mapeador do ListenBrainz (usar os que vieram do Navidrome, para não favorecer o ListenBrainz), remover os MBIDs e consultar o MusicBrainz (nome, duração e álbum) e o ListenBrainz (token de teste fornecido pelo usuário, nunca gravado). Registrar acerto, falso positivo, ambíguo, ausente e tempo por escuta. Medir também a ponte por ISRC: o getSong do Navidrome trouxe ISRC em 26 de 100 músicas, e o MusicBrainz tem consulta por ISRC (confirmar). Propor a regra de aceitação (campos, tolerância de duração, limiar de score) e a ordem recomendada das fontes. Entregar o relatório em backlog/evidencias/issue-NNN/ com contagens apenas: sem dados pessoais, sem tokens e sem URLs autenticadas.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Relatório com acerto, falso positivo, ambíguo, ausente e tempo por fonte, medidos com MBID de referência que não veio do mapeador do ListenBrainz
- [x] #2 Regra de aceitação proposta (campos conferidos, tolerância de duração, limiar de score) e ordem recomendada das fontes
- [x] #3 Ponte por ISRC medida: quantas escutas ganham MBID sem ambiguidade
- [x] #4 Relatório sem dados pessoais, tokens ou URLs autenticadas
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: ebrgr
created: 2026-11-26T16:44:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-215` na branch `spike/issue-215-estudo-medir-musicbrainz-por-nome` criada a partir de `main`.

---

author: dsh
created: 2026-11-26T18:23:00Z

---

Verificação: medição executada em 26/11/2026 com o script backlog/evidencias/issue-215/medir_issue_215.py. Amostra: 35 combinações distintas (histórico inteiro da conta: 1299 escutas, 39 com recording_mbid do cliente; nenhuma do Navidrome) — o alvo de 300 não foi possível por falta de dados, não por tempo. MusicBrainz por nome com a regra proposta (score >= 95, duração ±5 s, álbum e créditos conferidos): 28 acertos (80,0%), 0 falsos positivos, 6 ambíguas, 1 ausente, 0 erros HTTP; sem regra, o 1º resultado por score acerta só 31,4%. ListenBrainz lookup: 35/35 (100%), sem falsos positivos, e recuperou as 7 que o MusicBrainz não resolveu. Ponte por ISRC medida com ISRCs do próprio MusicBrainz (ISRC do Navidrome indisponível nesta rodada): 35/35 gravações com ISRC; 95,9% dos ISRCs levam a uma única gravação; 35/35 ganham MBID sem ambiguidade. Ordem recomendada: Navidrome → MusicBrainz por nome → ListenBrainz. Limites de taxa e User-Agent respeitados; 503 intermitentes do MusicBrainz contornados com espera e re-tentativas.

---
<!-- COMMENTS:END -->
