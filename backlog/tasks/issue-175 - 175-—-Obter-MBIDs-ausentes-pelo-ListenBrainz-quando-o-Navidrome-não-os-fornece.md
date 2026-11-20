---
id: ISSUE-175
title: 175 — Obter MBIDs ausentes pelo ListenBrainz quando o Navidrome não os fornece
status: Aberta
assignee: []
created_date: '2026-11-20T19:04:00Z'
updated_date: '2026-11-20T19:04:00Z'
labels:
  - ingestao
milestone: m-11
dependencies:
  - ISSUE-134
  - ISSUE-174
references:
  - backlog/evidencias/issue-134/PLANO.md
type: feature
ordinal: 188000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: decisão do usuário em 20/11/2026 sobre o plano da ISSUE-134. Última fonte de MBID de uma escuta, depois do MBID salvo ou recebido (ISSUE-134) e da API Subsonic do Navidrome (ISSUE-174): consultar o ListenBrainz (por exemplo o mapeamento de metadados/MBID do próprio ListenBrainz) quando o MBID continua ausente. Se o ListenBrainz estiver desligado, indisponível ou sem o dado, o MBID fica em branco, sem pendência e sem retry infinito. O resultado é afirmação declarada com fonte e data, não prova; não sobrescreve MBID já salvo ou recebido. Estudar na implementação o endpoint e os limites de uso do ListenBrainz e respeitar o relay e o token por perfil, sem enviar dado pessoal além do necessário (artista, faixa, álbum). Consulta fora do lock de escrita, com timeout, cache e limite de taxa.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Com o MBID ausente e o ListenBrainz disponível, a identidade recebe o MBID encontrado como afirmação declarada, com fonte e data, sem sobrescrever MBID existente
- [ ] #2 Com o serviço desligado, indisponível ou sem o dado, o MBID fica em branco sem pendência e sem retry infinito
- [ ] #3 Consulta fora do lock de escrita, com timeout, cache, limite de taxa e sem vazar dados além do necessário
<!-- AC:END -->
