---
id: ISSUE-129
title: '129 — Integração com o Orca: worktrees, preparação e orquestração com o Codex'
status: Em andamento
assignee:
  - '@ebrgr'
created_date: '2026-11-15T15:44:00Z'
updated_date: '2026-11-15T15:45:00Z'
labels:
  - processo
milestone: m-6
dependencies: []
type: chore
ordinal: 142000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: pedido do usuário em 15/11/2026, depois de instalar o Orca (orca.dev, 1.4.259). Objetivo: trabalhar tarefas em worktrees do Orca e orquestrar workers do Codex a partir do Claude, mantendo o fluxo issue-first. O Orca não lê o Backlog.md (só GitHub, Linear, GitLab, Jira): a ligação é pelo nome da branch <tipo>/issue-NNN-slug e pelo comentário da worktree.

Passos:
1. orca.yaml com scripts.setup: scripts/orca-setup.sh liga .agents/ e backlog/docs/estudos-locais/ (fora do git) a partir da pasta principal e confere as ferramentas.
2. scripts/issue.py: make issue.worktree ISSUE=N cria a worktree pelo Orca com a branch da tarefa e marca Em andamento; make issue.publish passa a aceitar branch que está numa worktree (remove a worktree limpa antes de publicar; recusa se houver mudanças não commitadas).
3. Pontos de conflito entre tarefas paralelas (CHANGELOG [Unreleased], referência de renderização, contagem no DEVELOPMENT.md): registrar e tratar em seguida.
4. Piloto: um worker Codex faz um estudo independente numa worktree enquanto o trabalho principal segue; registrar tempo, custo e qualidade.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 orca.yaml e scripts/orca-setup.sh preparam uma worktree nova com .agents e estudos locais
- [ ] #2 make issue.worktree cria a worktree pelo Orca com a branch da tarefa
- [ ] #3 make issue.publish publica uma tarefa feita numa worktree
- [ ] #4 Piloto com um worker Codex concluído e avaliado
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: ebrgr
created: 2026-11-15T15:45:00Z

---

Início: branch `chore/issue-129-integracao-com-o-orca-worktrees` criada a partir de `main`.

---
<!-- COMMENTS:END -->
