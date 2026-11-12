---
id: ISS-004
title: 004 — Confirmar contratos de paginação, filtros e datas
status: Done
assignee: []
created_date: '2026-11-12T23:39:00Z'
labels:
  - ui
milestone: m-0
dependencies: []
references:
  - docs-dev/issues/004-contratos-paginas-solved.md
type: feature
ordinal: 4000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
> Importada de [docs-dev/issues/004-contratos-paginas-solved.md](../../docs-dev/issues/004-contratos-paginas-solved.md) — numeração local 004.

Status: **fechada e resolvida em 04/11/2026**, após o núcleo da issue 003.
Issue local, sem publicação. Plano: [roadmap 2](../../docs-dev/ROADMAP_02_PAGINAS.md).

## Trabalho

- Reproduzir o sombreamento de `params.slug` em `ChartLayout.updateParams`.
- Unificar a paginação HTML em base 1 e validar limite, IDs e overflow.
- Garantir que rótulo e filtro ativo representem o intervalo consultado.
- Cobrir semana ISO, períodos móveis, fuso forçado/query/cookie e perfil vazio.
- Preservar o rank do backend, incluindo empates, e verificar desempate do histórico.
- Distinguir página além do fim de ausência de todo o histórico.
- Confirmar propagação de erros de contagem e identificar escutas por faixa/segundo.
- Preservar os contratos da API existente ao extrair regras compartilhadas.

## Aceite

Casos de regressão concretos para os pontos confirmados, incluindo filtros sem
perda do slug e testes efetivos de datas. Nenhuma correção funcional foi feita
na criação desta issue. Não alterar migrações para corrigir apresentação.

## Marco da implementação

Normalização HTML, paginação em base 1, desempate por faixa, erros de contagem
e metadados de intervalo da API implementados. Testes Go de páginas/datas/SQLite
e typecheck e testes React passaram. A integração do modo misto segue na 005.

## Encerramento

Corrigidos slug nos filtros React, base 1, validação HTML e limites/overflow;
intervalos e rótulos compartilham limites resolvidos. Testes efetivos cobrem ISO,
bissexto, DST, fusos, empates e cancelamento da contagem. Histórico usa desempate
por faixa e identidade composta na exclusão; React revalida dados após sucesso.
API conserva seus campos e recebeu `range` aditivo. Nenhuma migração nova.

Validação e capturas: [relatório do roadmap 2](../../docs-dev/comparisons/roadmap-02/README.md).
`make check ui.check`, build React/Go, testes de corrida focados e validações
HTTP/multiusuário passaram. Sem commit ou publicação.
<!-- SECTION:DESCRIPTION:END -->
