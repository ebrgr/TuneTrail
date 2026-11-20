---
id: ISSUE-177
title: 177 — Reduzir leituras históricas repetidas na Home e no Rewind
status: Aberta
assignee: []
created_date: '2026-11-20T22:56:00Z'
updated_date: '2026-11-20T22:56:00Z'
labels:
  - api
  - banco
milestone: m-13
dependencies:
  - ISSUE-135
references:
  - backlog/evidencias/issue-092/BENCHMARK.md
type: enhancement
ordinal: 190000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: ISSUE-092 (aprovada pelo usuário em 20/11/2026). Em 1 milhão de escutas sintéticas (3 perfis de cerca de um terço), p95 de 2,607 s na Home em «Tudo», 1,214 s na Home semanal e 1,311 s no Rewind HTML. Um índice experimental reduziu a Home «Tudo» para 2,036 s (-22%) e a semanal para 0,902 s (-26%) com +17,85 MB no banco de 1 milhão, mas a Home «Tudo» continuou acima de 2 s; a seleção semanal também lê o painel histórico (internal/webui/home.go:27–33). Escopo: depois da troca dos leitores por ocorrência (ISSUE-135), medir de novo, isolar o custo do painel histórico, das novidades (new_tracks calcula MIN antes de filtrar) e das consultas repetidas, e só então adotar otimizações (índices, cache ou materialização) com medição antes e depois, custo de escrita na ingestão e de criação do índice. Não indexar o heatmap nem o histórico legado (as medições não justificam).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Nova medição com 1 milhão de escutas depois da ISSUE-135, isolando painel histórico, novidades e consultas repetidas
- [ ] #2 Otimizações adotadas só com ganho medido e custo de escrita e de disco registrados; Home em «Tudo» com p95 abaixo da meta aprovada
<!-- AC:END -->
