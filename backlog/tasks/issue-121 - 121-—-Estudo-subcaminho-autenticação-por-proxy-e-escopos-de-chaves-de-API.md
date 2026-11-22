---
id: ISSUE-121
title: '121 — Estudo: subcaminho, autenticação por proxy e escopos de chaves de API'
status: Aberta
assignee: []
created_date: '2026-11-14T22:54:00Z'
updated_date: '2026-11-22T20:05:00Z'
labels:
  - operacao
milestone: m-15
dependencies: []
type: spike
ordinal: 130000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: estudo ISSUE-114 (doc-034). Implantação: servir em subcaminho (Koito #143), desativar login quando há autenticação por proxy/forward-auth (#339) e escopos para chaves de API, separando rotas só de sessão (roadmap Koito #284, 0.4.0). Avaliar impacto em cookies, CSRF, links absolutos e perfis públicos (ISSUE-100).
<!-- SECTION:DESCRIPTION:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-22T20:05:00Z

---

Do PR gabehf/Koito#382 (22/11/2026): uma API de leitura do ListenBrainz (/user/{u}/listens, playing-now, listen-count, /stats/user/{u}/artists|releases|recordings|listening-activity, feedback) exige chaves com escopo de leitura separado do de ingestão; hoje a chave só ingere. Considerar no estudo de escopos: ingest, read-stats, read-listens e visibilidade do perfil (privado, público). Rotas e formatos do PR servem de referência; o ranking por gravação e por lançamento (ISSUE-135) já casa com recordings e releases. Implementação da API de leitura: nova issue depois desta e da 118.

---
<!-- COMMENTS:END -->
