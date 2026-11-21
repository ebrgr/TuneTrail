---
id: ISSUE-183
title: 183 — Edição, exclusão, fusão e exportação por ocorrência (recorte da 135)
status: Em andamento
assignee:
  - '@ebrgr'
created_date: '2026-11-21T22:02:00Z'
updated_date: '2026-11-21T23:04:00Z'
labels:
  - catalogo
milestone: m-13
dependencies:
  - ISSUE-135
references:
  - backlog/evidencias/issue-183/VERIFICATION.md
  - backlog/evidencias/issue-183/REVIEW-FOLLOWUP.md
type: feature
ordinal: 196000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Recorte da ISSUE-135 (decisão do usuário em 21/11/2026). Edição, exclusão, fusão e exportação/reimportação passam a operar por ocorrência (doc-036, doc-039, decision-008): editar ou remover um lançamento não altera outro; fusão não deduplica por gravação/segundo; exportação versionada com cursor por event_id preserva eventos simultâneos, ocorrência e proveniência. Entradas obrigatórias da ISSUE-133 (comentário #2 da 135): merge reversível de gravações para ocorrências equivalentes em lançamentos distintos, reimportação e compatibilidade integrada do corte, comparação HTTP do AC3 versionada e reaplicada aos leitores ativados.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Editar ou remover um lançamento não muda outro; fusão não deduplica eventos por gravação/segundo
- [x] #2 Exportar e reimportar preserva ocorrência e proveniência; exportação versionada com cursor por event_id
- [x] #3 Duas escutas simultâneas de perfis distintos sobrevivem; planos de consulta medidos em fixtures grandes
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: ebrgr
created: 2026-11-21T22:28:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-183` na branch `feat/issue-183-edicao-exclusao-fusao-e-exportacao` criada a partir de `main`.

---

author: Codex
created: 2026-11-21T22:49:00Z

---

Implementação verificada: edição e exclusão por ocorrência, fusão de gravações reversível e exportação/reimportação v2 idempotente por perfil. make check passou (13/13 jornadas de ponta a ponta), a comparação HTTP com a v0.55 confirmou 39 respostas equivalentes após a migração e a ativação, e o EXPLAIN confirmou paginação indexada em 300 mil eventos. Referência ISSUE-075 regenerada.

---

author: advisor
created: 2026-11-21T22:50:00Z

---

Reviewer: sem BUG confirmado; 4 RISCOs. Advisor (effort high): REPROVADO COM RESSALVAS — vazamento entre perfis na restauração de arquivo (archive.go mapID, namespace local) a corrigir; melhorias: cursor de export vs escritas concorrentes, fusão em transação longa. Ver backlog/evidencias/issue-183/PARECER-ADVISOR.md e PARECER-REVIEWER.md.

---

author: Codex
created: 2026-11-21T22:59:00Z

---

Correções verificadas: o importador restringe o reuso de ocorrências locais pelo dono; a exportação fixa o teto na primeira página; o desfazer de fusões expira em 30 dias. Na fixture de 5.001 ocorrências e 10.000 recibos, fusão e desfazer levaram 184 ms e 201 ms, com todas as revisões preservadas. make check passou (13/13 jornadas). Referência ISSUE-075 regenerada e 39 respostas HTTP equivalentes após migração para v15 e ativação.

---

author: claude
created: 2026-11-21T23:04:00Z

---

Reviewer (reauditoria do commit 99c51fa): sem BUG; falha de segurança do Advisor corrigida e testada; cursor de export com teto, fusão 184 ms, desfazer com expiração. Ver PARECER-REVIEWER.md.

---
<!-- COMMENTS:END -->
