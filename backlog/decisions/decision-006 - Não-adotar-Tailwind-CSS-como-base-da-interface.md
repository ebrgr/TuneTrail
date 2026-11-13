---
id: decision-006
title: Não adotar Tailwind CSS como base da interface
date: '2026-11-12T23:40:00Z'
status: proposed
---
## Context
O CSS atual tem larguras fixas em px, nove breakpoints e dois estilos de escrita (ISSUE-058).


## Decision
Reorganizar o CSS próprio com tokens, `@layer` e container queries (opção D); reavaliar Tailwind somente após prova de conceito.


## Consequences
Sem etapa de build nova; a decisão final depende da avaliação do usuário na ISSUE-058.

