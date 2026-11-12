---
id: ISS-047
title: 047 — Corrigir curva do gráfico de escutas
status: Done
assignee: []
created_date: '2026-11-12T23:39:00Z'
labels:
  - ui
milestone: m-6
dependencies: []
references:
  - docs-dev/issues/047-correcao-curva-grafico-solved.md
type: bug
ordinal: 47000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
> Importada de [docs-dev/issues/047-correcao-curva-grafico-solved.md](../../docs-dev/issues/047-correcao-curva-grafico-solved.md) — numeração local 047.

**Status:** Fechada e aprovada pelo usuário em 08/11/2026

## Problema

A interpolação cúbica natural pode ultrapassar os valores dos pontos, inclusive a linha zero. O recorte do SVG então oculta partes da linha e distorce o preenchimento do gráfico de interesse.

## Objetivo e aceite

- Substituir a interpolação por uma curva contínua que permaneça entre os valores de cada par de pontos.
- Preservar valores, intervalos, escala, cores, degradê e demais comportamentos do gráfico.
- Testar zeros consecutivos, picos, alternâncias e sequências constantes, assegurando que cada trecho fique dentro dos limites dos pontos.
- Atualizar o changelog e executar os testes pertinentes da interface Go.

## Resultado

A curva usa segmentos Bézier com tangentes horizontais nos pontos, mantendo cada trecho entre os valores dos extremos. As contagens, a escala, o degradê e o recorte foram preservados. Os testes de `internal/webui` passaram, incluindo zeros consecutivos, picos e limites numéricos. A visualização da faixa específica depende de conferência na instância do usuário.

## Recomendação futura

Verificar a interpolação para fazer as curvas mais redondas, mantendo os trechos dentro dos limites dos pontos.
<!-- SECTION:DESCRIPTION:END -->
