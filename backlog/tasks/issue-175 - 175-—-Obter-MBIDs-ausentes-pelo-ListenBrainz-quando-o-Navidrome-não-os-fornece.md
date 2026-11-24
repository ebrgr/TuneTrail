---
id: ISSUE-175
title: 175 — Obter MBIDs ausentes pelo ListenBrainz quando o Navidrome não os fornece
status: Aguardando aceite
assignee:
  - '@ebrgr'
created_date: '2026-11-20T19:04:00Z'
updated_date: '2026-11-24T18:20:00Z'
labels:
  - ingestao
milestone: m-11
dependencies:
  - ISSUE-134
  - ISSUE-174
references:
  - backlog/evidencias/issue-134/PLANO.md
  - backlog/evidencias/issue-175/FLUXO.md
  - backlog/evidencias/issue-175/PARECER.md
type: feature
ordinal: 188000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: decisão do usuário em 20/11/2026 sobre o plano da ISSUE-134. Última fonte de MBID de uma escuta, depois do MBID salvo ou recebido (ISSUE-134) e da API Subsonic do Navidrome (ISSUE-174): consultar o ListenBrainz (por exemplo o mapeamento de metadados/MBID do próprio ListenBrainz) quando o MBID continua ausente. Se o ListenBrainz estiver desligado, indisponível ou sem o dado, o MBID fica em branco, sem pendência e sem retry infinito. O resultado é afirmação declarada com fonte e data, não prova; não sobrescreve MBID já salvo ou recebido. Estudar na implementação o endpoint e os limites de uso do ListenBrainz e respeitar o relay e o token por perfil, sem enviar dado pessoal além do necessário (artista, faixa, álbum). Consulta fora do lock de escrita, com timeout, cache e limite de taxa.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Com o MBID ausente e o ListenBrainz disponível, a identidade recebe o MBID encontrado como afirmação declarada, com fonte e data, sem sobrescrever MBID existente
- [x] #2 Com o serviço desligado, indisponível ou sem o dado, o MBID fica em branco sem pendência e sem retry infinito
- [x] #3 Consulta fora do lock de escrita, com timeout, cache, limite de taxa e sem vazar dados além do necessário
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: codex
created: 2026-11-24T11:51:00Z

---

Decisão do usuário em 24/11/2026: buscar MBID de gravação e de artista pelo ListenBrainz apenas quando o respectivo identificador não estiver disponível nos metadados recebidos e continuar ausente após o fallback Navidrome. Preservar identificadores já recebidos; a busca de outros metadados segue regras próprias da ISSUE-055.

---

author: ebrgr
created: 2026-11-24T17:37:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-175` na branch `feat/issue-175-obter-mbids-ausentes-pelo-listenbrainz` criada a partir de `main`.

---

author: codex
created: 2026-11-24T17:53:00Z

---

Implementado o fallback autenticado do ListenBrainz por gravação e crédito, com autorização do relay oficial no perfil do recibo, proveniência por escuta e reimportação v2. A verificação com servidor simulado cobriu precedência, isolamento, ausência de token, limites, falhas e conflitos concorrentes. O make check passou com 13 percursos E2E, os testes focados passaram com o detector de corrida e o grafo AST foi atualizado. O endpoint fornece um candidato canônico, sem alternativas para distinguir gravações com a mesma combinação textual.

---

author: codex
created: 2026-11-24T18:03:00Z

---

A revisão identificou a compactação prematura dos MBIDs por crédito. O adaptador preserva os slots, a admissão usa digest v3 com projeções compatíveis para repetições v1 e v2 e a exportação mantém a procedência por crédito. A regressão parte do JSON e cobre lacunas no início, no meio e no fim, com Subsonic, ListenBrainz e restauração v2. O make check passou com 13 percursos E2E e a regressão passou com o detector de corrida.

---

author: ebrgr
created: 2026-11-24T18:20:00Z

---

Verificação: fallback por perfil e por crédito, posições vazias, precedência, reenvio com digest anterior, relay e arquivo v2 testados. Revisão aprovada e make check passou.

---
<!-- COMMENTS:END -->
