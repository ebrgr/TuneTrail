---
id: decision-007
title: Registrar issues, estudos, roadmaps e decisões no Backlog.md
date: '2026-11-12T23:40:00Z'
status: accepted
---
## Context
Issues e estudos ficavam em arquivos Markdown soltos em docs-dev, sem quadro nem consulta estruturada. Foram avaliados kanban-md, gitplan, Aver, gh-issue-sync e Backlog.md.


## Decision
Usar Backlog.md (instalado via Homebrew) com o diretório `backlog/` na raiz. Estudos privados ficam em `backlog/docs/estudos-locais/`, excluído do Git por `.git/info/exclude`.


## Consequences
O GitHub continua sendo o destino das issues publicadas (issue-first); a publicação é um passo separado que depende de autorização do usuário.

