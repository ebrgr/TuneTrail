---
id: ISSUE-062
title: 062 — Reconciliar registros do projeto e higiene do repositório
status: Em andamento
assignee:
  - '@ebrgr'
created_date: '2026-11-14T00:38:00Z'
updated_date: '2026-11-14T01:20:00Z'
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
- [ ] #1 doc-001 e doc-016 identificados como históricos, com link para o roadmap vigente
- [ ] #2 ISSUE-046 com critérios marcados somente por evidência e lacunas registradas
- [ ] #3 doc-010 §1 reflete o estado da versão 0.8
- [ ] #4 git status limpo após build e testes locais (artefatos ignorados)
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: ebrgr
created: 2026-11-14T01:20:00Z

---

Início: branch `docs/issue-062-reconciliar-registros-do-projeto-e` criada a partir de `main`.

---
<!-- COMMENTS:END -->
