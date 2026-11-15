---
id: ISSUE-129
title: '129 — Integração com o Orca: worktrees, preparação e orquestração com o Codex'
status: Em andamento
assignee:
  - '@ebrgr'
created_date: '2026-11-15T15:44:00Z'
updated_date: '2026-11-15T16:03:00Z'
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

author: claude
created: 2026-11-15T16:03:00Z

---

Quadro no Orca: o 'backlog browser --port 6420 --no-open' passou a rodar num terminal 'Backlog' do próprio Orca, no checkout principal, e a aba do navegador aponta para http://127.0.0.1:6420/board. Remoto 'upstream' (gabehf/Koito) renomeado para 'koito' (configuração local; nada no projeto usava o nome), porque o Orca trata 'upstream' como o repositório principal e listava as issues do Koito. O GitHub ainda registra ebrgr/TuneTrail como fork do Koito e as issues estão desligadas na origin; as issues do projeto continuam no Backlog.md.

---
<!-- COMMENTS:END -->
