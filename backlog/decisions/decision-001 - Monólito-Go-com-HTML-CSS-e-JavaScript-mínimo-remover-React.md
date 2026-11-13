---
id: decision-001
title: Monólito Go com HTML, CSS e JavaScript mínimo; remover React
date: '2026-11-12T23:40:00Z'
status: accepted
---
## Context
O Koito separava backend Go e frontend React. A proposta do fork é simplificar a entrega e reduzir dependências externas (ROADMAP, doc-001).


## Decision
Renderizar todas as páginas no servidor Go com templates incorporados, CSS e scripts pequenos; React retirado após comparação visual e aceite (ISSUE-022–ISSUE-030).


## Consequences
Executável único sem Node em produção; fluxos principais funcionam sem JavaScript. Interações dinâmicas futuras devem justificar cada script (ISSUE-056).

