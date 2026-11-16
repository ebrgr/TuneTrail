---
id: ISSUE-146
title: >-
  146 — App nativo iOS e Android de leitura (estatísticas, histórico, tocando
  agora)
status: Aberta
assignee: []
created_date: '2026-11-16T01:05:00Z'
labels:
  - api
  - ui
milestone: m-12
dependencies:
  - ISSUE-145
  - ISSUE-076
documentation:
  - doc-038
priority: low
type: feature
ordinal: 159000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: estudo doc-038. Futuro, só depois do contrato e da autenticação (ISSUE-145) e de avaliar o PWA (ISSUE-076): app para iOS e Android que lê um servidor TuneTrail próprio e mostra estatísticas, tops, histórico, tocando agora e busca. Escolha da tecnologia (um framework único ou dois apps nativos) é a primeira decisão da tarefa. Fora do escopo: gestão do catálogo (fica na web) e captura de escutas (ISSUE-147).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Tecnologia escolhida e justificada, depois de avaliar o PWA da ISSUE-076
- [ ] #2 App mostra estatísticas, tops, histórico, tocando agora e busca de um servidor informado pelo usuário
- [ ] #3 Autenticação conforme o contrato da ISSUE-145, com revogação do dispositivo
- [ ] #4 Publicação nas lojas ou distribuição de teste definida
<!-- AC:END -->
