---
id: ISSUE-135
title: >-
  135 — Consultas, edição, exclusão e exportação por ocorrência, com ranking por
  gravação ou lançamento
status: Em andamento
assignee:
  - '@ebrgr'
created_date: '2026-11-15T16:43:00Z'
updated_date: '2026-11-21T22:21:00Z'
labels:
  - catalogo
  - ui
milestone: m-13
dependencies:
  - ISSUE-133
  - ISSUE-134
  - ISSUE-092
references:
  - backlog/evidencias/issue-135/planos-ranking.md
  - backlog/evidencias/issue-135/parecer-reviewer.md
documentation:
  - doc-036
  - doc-039
type: feature
ordinal: 148000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Recorte 5 do estudo da ISSUE-116 (doc-036). Consultas, edição, fusão, exclusão e exportação passam a usar ocorrências. Decisão do usuário: a tela de faixas mais ouvidas ganha uma chave 'por gravação' / 'por lançamento', com padrão por gravação.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Chave 'por gravação' / 'por lançamento' no ranking de faixas, padrão por gravação; soma por lançamento igual ao total por gravação
- [x] #2 Rankings e novidades por perfil seguem o contrato da ISSUE-132
- [x] #3 Duas escutas simultâneas de perfis distintos contam ambas no ranking; planos de consulta do ranking medidos em fixture grande
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: codex
created: 2026-11-16T17:06:00Z

---

Contrato da ISSUE-132 aceito (decision-008). Reconciliação por lançamento inclui o balde sem lançamento; pendências não contam antes de resolução. Exportação versionada e cursor por event_id preservam eventos simultâneos; edição/exclusão/fusão não deduplicam por gravação/segundo. Manter filtros de perfil e URLs antigas de ocorrência.

---

author: codex
created: 2026-11-19T19:06:00Z

---

Entradas obrigatórias da ISSUE-133: merge reversível de gravações para ocorrências equivalentes em lançamentos distintos (faixas com MBID nulo viram gravações separadas e o ranking fica dividido até reconciliação); confirmar reimportação e compatibilidade integrada do corte, pois AC6/AC7 da 133 usam primitivas SQL em fixture pós-corte. Versionar a comparação HTTP do AC3 e reaplicá-la aos leitores ativados.

---

author: claude
created: 2026-11-21T22:02:00Z

---

Escopo dividido em 21/11/2026 (decisão do usuário): edição, exclusão, fusão e exportação/reimportação (antigos AC #3 e #4) foram para a ISSUE-183. Esta issue fica com a chave por gravação/lançamento e os rankings por perfil. Escuta sem lançamento conta no balde 'Lançamento desconhecido'. Advisor (effort high) depois da implementação.

---

author: ebrgr
created: 2026-11-21T22:02:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-135` na branch `feat/issue-135-consultas-edicao-exclusao-e-exportacao` criada a partir de `main`.

---

author: codex
created: 2026-11-21T22:11:00Z

---

Verificação: ranking por gravação e por lançamento, soma com o balde sem lançamento, isolamento das novidades por perfil e escutas no mesmo segundo cobertos por testes SQLite e HTTP. Oito planos medidos em fixture de 600.004 escutas com índices existentes. make check passou com 13/13 jornadas E2E. Referência ISSUE-075 regenerada. Revisão local sem bloqueios.

---

author: claude
created: 2026-11-21T22:21:00Z

---

Reviewer: sem BUG; 3 RISCOs (desempenho por lançamento, contadores por gravação, histórico lendo occurrences). Ver backlog/evidencias/issue-135/parecer-reviewer.md. Testes -race OK. Advisor (effort high) pendente, a pedido do usuário.

---
<!-- COMMENTS:END -->
