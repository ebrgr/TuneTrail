---
id: ISSUE-174
title: 174 — Obter MBIDs ausentes pela API Subsonic do Navidrome
status: Aberta
assignee: []
created_date: '2026-11-20T19:04:00Z'
updated_date: '2026-11-20T19:04:00Z'
labels:
  - ingestao
milestone: m-11
dependencies:
  - ISSUE-134
references:
  - backlog/evidencias/issue-134/PLANO.md
type: feature
ordinal: 187000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: decisão do usuário em 20/11/2026 sobre o plano da ISSUE-134 (identidade por MBID). Ordem de fontes do MBID de uma escuta: (1) o MBID já salvo no registro ou recebido junto da escuta; (2) quando ausente, consultar a API Subsonic do Navidrome do usuário (getSong/getAlbum/search3; extensões OpenSubsonic quando existirem) e ler os MBIDs de faixa, gravação, álbum e artista que o servidor expõe; (3) quando ausente, a ISSUE-175 consulta o ListenBrainz; (4) se o dado estiver ausente ou o serviço estiver desligado ou indisponível, deixar o MBID em branco, sem pendência nem retry infinito, e resolver a identidade pelos metadados. O MBID obtido de serviço externo é afirmação declarada (não prova): grava-se a fonte e a data, e a promoção a validado segue o contrato da 134. Reaproveitar os parâmetros Subsonic já usados por capas (KOITO_SUBSONIC_PARAMS, internal/images/subsonic.go) e a ISSUE-055/088 (metadados e fonte Subsonic). A consulta ocorre fora do lock de escrita, com timeout, cache e limite de taxa, e nunca bloqueia a admissão. Observação: o formulário de credenciais do Subsonic (user, token, salt) é decisão separada do usuário, ainda pendente.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Com o Subsonic configurado e o MBID ausente, a identidade recebe os MBIDs expostos pelo Navidrome, com fonte e data registradas como declaração
- [ ] #2 Com o serviço desligado, indisponível ou sem o dado, o MBID fica em branco sem pendência, sem retry infinito e sem bloquear a admissão
- [ ] #3 Consulta fora do lock de escrita, com timeout, cache e limite de taxa; credenciais nunca aparecem em log
<!-- AC:END -->
