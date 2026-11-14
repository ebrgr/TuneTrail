---
id: ISSUE-062
title: 062 — Reconciliar registros do projeto e higiene do repositório
status: Done
assignee:
  - '@ebrgr'
created_date: '2026-11-14T00:38:00Z'
updated_date: '2026-11-14T01:25:00Z'
labels:
  - docs
  - processo
milestone: m-9
dependencies: []
priority: high
type: docs
ordinal: 62000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: doc-017 §Organização documental, doc-010 §1.3 e §8.

Passos:
1. Marcar como históricos os trechos do doc-001 (ROADMAP antigo) que ainda citam modo Go opcional, React disponível e a 026 planejada; apontar para o novo roadmap.
2. Marcar o doc-016 (estudo inicial de pendências) como fotografia histórica, com link para o estado atual.
3. ISSUE-046: marcar somente os critérios exercitados e registrar falhas sistemáticas e restauração completa como lacunas reais (seguem na tarefa de testes de recuperação).
4. Atualizar doc-010 §1 com o estado pós-0.8 (commits e aceites pendentes resolvidos).
5. Higiene: garantir que binários locais (`tunetrail`, `koito`) e diretórios temporários estão no .gitignore; remover restos não versionados com aprovação do usuário.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 doc-001 e doc-016 identificados como históricos, com link para o roadmap vigente
- [x] #2 ISSUE-046 com critérios marcados somente por evidência e lacunas registradas
- [x] #3 doc-010 §1 reflete o estado da versão 0.8
- [x] #4 git status limpo após build e testes locais (artefatos ignorados)
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: ebrgr
created: 2026-11-14T01:20:00Z

---

Início: branch `docs/issue-062-reconciliar-registros-do-projeto-e` criada a partir de `main`.

---

author: ebrgr
created: 2026-11-14T01:23:00Z

---

Verificação: (1) doc-001 e doc-016 marcados como históricos, com 7 notas de estado atual no doc-001 e link para o doc-032; (2) ISSUE-046 com critérios 1, 2, 3 e 7 marcados após reexecutar hoje os 13 testes TestBootstrap*, scripts/validate_migration_030.py e make multiuser.check (todos aprovados com o binário 0.8); critérios 4, 5, 6 e 8 parciais, lacunas acrescentadas como critérios 5–7 da ISSUE-064; (3) doc-010 §1 anotado com o estado 0.8 e as tarefas que cobrem cada ponto de atenção; (4) make check e build sem arquivos novos no git status; .gitignore já cobre binários, __pycache__ e .DS_Store. Restos locais ignorados (cmd/api/__debug_bin4125864029, 26 MB; .DS_Store; scripts/__pycache__) aguardam aprovação do usuário para remoção. Links do backlog: 933 válidos. Commit 3d967c4.

---

author: claude
created: 2026-11-14T01:25:00Z

---

Aceite: aprovada pelo usuário em 13/11/2026. Restos locais ignorados removidos com autorização.

---

author: claude
created: 2026-11-14T01:25:00Z

---

Publicação: integrada ao main por fast-forward sem versão nem tag, a pedido do usuário (tarefa de documentação, conforme o doc-032).

---
<!-- COMMENTS:END -->
