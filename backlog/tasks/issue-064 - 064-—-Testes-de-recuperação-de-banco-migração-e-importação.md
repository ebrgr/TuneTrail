---
id: ISSUE-064
title: 064 — Testes de recuperação de banco, migração e importação
status: Aberta
assignee: []
created_date: '2026-11-14T00:38:00Z'
updated_date: '2026-11-14T01:22:00Z'
labels:
  - banco
  - validacao
milestone: m-9
dependencies:
  - ISSUE-062
priority: high
type: chore
ordinal: 64000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: doc-017 (prioridade alta) — ISSUE-046 e ISSUE-020/021 não exercitaram falhas sistemáticas.

Passos:
1. Migração `koito.db` → `tunetrail.db`: interrupção em cada fase, falta de espaço, WAL pendente, destino existente, falha na publicação; tudo em diretórios descartáveis.
2. Importação: reinício no meio, arquivo truncado, falha de disco; confirmar que escutas já gravadas são preservadas.
3. Restauração: ensaio completo a partir do backup gerado pela migração.
4. Registrar contratos que continuam como estão (importação parcial, exportação dinâmica) e o que exigiria funcionalidade nova.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Testes automatizados de interrupção e falha para cada fase da migração
- [ ] #2 Testes de reinício e falha de disco durante importação, sem perda de escutas gravadas
- [ ] #3 Ensaio de restauração documentado em backlog/evidencias
- [ ] #4 Lacunas restantes registradas como tarefas ou limitações aceitas
- [ ] #5 Teste de recusa de versão de esquema desconhecida (guarda em bootstrap.go:124 sem cobertura hoje)
- [ ] #6 Rollback exercitado: versão anterior iniciada a partir do koito.db legado e do .bak com fixture descartável
- [ ] #7 DEPLOYMENT.md com procedimento passo a passo para recuperar migração interrompida a partir do .bak
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-14T01:22:00Z

---

Lacunas da ISSUE-046 reconciliadas na ISSUE-062: critérios 4 (versão desconhecida sem teste), 5 (injeção de falhas e interrupção por fase), 6 (rollback não exercitado) e 8 (procedimento de recuperação). Os três primeiros critérios desta tarefa já cobrem a injeção de falhas; acrescentados critérios para os demais.

---
<!-- COMMENTS:END -->
