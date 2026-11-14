---
id: decision-006
title: Não adotar Tailwind CSS como base da interface
date: '2026-11-12T23:40:00Z'
status: accepted
---
## Context
O CSS atual tem larguras fixas em px, nove breakpoints e dois estilos de escrita (ISSUE-058).


## Decision
Reorganizar o CSS próprio com tokens, `@layer` e container queries (opção D); reavaliar Tailwind somente após prova de conceito.


## Consequences
Sem etapa de build nova. Aceita pelo usuário em 13/11/2026: Tailwind não será adotado; a reorganização segue em ISSUE-061 (doc-031).

