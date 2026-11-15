---
id: ISSUE-130
title: 130 — Estudo de um plugin do Backlog.md para o Orca
status: Done
assignee:
  - '@ebrgr'
created_date: '2026-11-15T16:06:00Z'
updated_date: '2026-11-15T16:17:00Z'
labels:
  - processo
dependencies: []
documentation:
  - doc-037
type: spike
ordinal: 143000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Estudar a viabilidade e o plano de execução de um plugin do Orca que mostre e opere as tarefas do Backlog.md. Fase 1 (painel de leitura): barra lateral com as tarefas agrupadas por status (Aberta, Em andamento, Aguardando aceite), busca e botão para abrir o arquivo da tarefa. Fase 2 (painel com ações): botões 'Iniciar em worktree (Claude ou Codex)' e 'Despachar worker', e contador de tarefas Aguardando aceite na barra de status. Depois, avaliar a publicação do plugin no catálogo da comunidade.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Mapear a API de plugins do Orca (manifesto, painéis, barra de status, comandos, permissões, sandbox, testes)
- [x] #2 Definir como o plugin lê o Backlog.md (CLI --json ou arquivos) sem depender do TuneTrail
- [x] #3 Plano de execução em tarefas para as fases 1 e 2, com estimativa
- [x] #4 Requisitos e passos para publicar no catálogo da comunidade, sem publicar
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: ebrgr
created: 2026-11-15T16:06:00Z

---

Início: branch `spike/issue-130-estudo-de-um-plugin-do` criada a partir de `main`.

---

author: ebrgr
created: 2026-11-15T16:10:00Z

---

Verificação: estudo doc-037 com API de plugins do Orca 1.4.259 conferida (referências do orca-plugin-authoring, plugin-host-entry.js sem modelo de permissões do Node), saída backlog --json/--watch medida (~70 ms), PATH do app conferido (/usr/bin:/bin:/usr/sbin:/sbin), catálogo da comunidade consultado (299 plugins, nenhum do Backlog.md). Nada publicado.

---

author: claude
created: 2026-11-15T16:17:00Z

---

Aceite do usuário em 15/11/2026. Decisões: (1) o plugin terá repositório próprio, ebrgr/backlog-md-orca, com o próprio Backlog.md; toda a implementação segue lá; (2) projeto em inglês; (3) tudo separado do TuneTrail, com o checkout local em ~/Projects/backlog-md-orca. Esta tarefa fica encerrada com o estudo doc-037.

---
<!-- COMMENTS:END -->
