---
id: ISSUE-178
title: 178 — Revisar o gráfico Interest over time frente ao Koito legado
status: Aberta
assignee: []
created_date: '2026-11-21T13:58:00Z'
labels:
  - ui
dependencies: []
references:
  - internal/webui/interest.go
  - internal/db/sqlite/interest.go
  - >-
    backlog/docs/roadmaps/doc-005 -
    Roadmap-3-—-artistas-álbuns-faixas-e-retrospectiva.md
type: enhancement
ordinal: 191000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Estudo comparativo (2026-11-21) do gráfico Interest over time do TuneTrail com o Koito legado (gabehf/Koito, branch main: client/app/components/InterestGraph.tsx e internal/db/sqlite/interest.go). Comparação feita só por leitura de código; nenhuma captura de tela lado a lado.

Diferenças visuais: (1) curva — legado usa recharts type="natural" (spline cúbico suave, pode ultrapassar); TuneTrail usa boundedCurve em internal/webui/interest.go, com tangentes horizontais (platôs, sem overshoot); (2) escala vertical — legado usa o eixo automático do recharts; TuneTrail normaliza pelo pico; (3) extras do TuneTrail: pontos, tooltip e tabela Interest values (ISSUE-075).

Diferenças de cálculo (internal/db/sqlite/interest.go): (1) fronteiras — legado usa totalSeconds/n inteiro e perde o resto, TuneTrail usa fronteiras exatas até agora; (2) atribuição — legado usa (ts-start)*n/(totalSeconds+1), que não bate com os rótulos (exemplo 100→110, 3 baldes, em backlog/docs/roadmaps doc-005); TuneTrail atribui pelas mesmas fronteiras dos rótulos; (3) histórico de um único instante (span 0) — legado põe tudo no primeiro balde, TuneTrail no último (provável efeito colateral, a confirmar com teste); (4) entradas inválidas — TuneTrail valida o máximo de 512 baldes, filtra escutas futuras e rejeita timestamps fora do intervalo.

A decidir com o usuário: manter a curva atual ou adotar spline natural limitado a 0; manter ou igualar a escala vertical; corrigir o caso de instante único para cair no primeiro balde.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Decidir, com o usuário, quais diferenças com o Koito legado devem ser mantidas ou igualadas
- [ ] #2 Se a curva mudar, comparar capturas lado a lado com o legado
- [ ] #3 Cobrir com teste o histórico de um único instante
<!-- AC:END -->
