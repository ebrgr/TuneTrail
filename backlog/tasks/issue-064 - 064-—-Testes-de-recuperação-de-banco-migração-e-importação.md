---
id: ISSUE-064
title: 064 — Testes de recuperação de banco, migração e importação
status: Done
assignee:
  - '@ebrgr'
created_date: '2026-11-14T00:38:00Z'
updated_date: '2026-11-14T01:48:00Z'
labels:
  - banco
  - validacao
milestone: m-9
dependencies:
  - ISSUE-062
references:
  - backlog/evidencias/recuperacao/ISSUE_064.md
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
- [x] #1 Testes automatizados de interrupção e falha para cada fase da migração
- [x] #2 Testes de reinício e falha de disco durante importação, sem perda de escutas gravadas
- [x] #3 Ensaio de restauração documentado em backlog/evidencias
- [x] #4 Lacunas restantes registradas como tarefas ou limitações aceitas
- [x] #5 Teste de recusa de versão de esquema desconhecida (guarda em bootstrap.go:124 sem cobertura hoje)
- [x] #6 Rollback exercitado: versão anterior iniciada a partir do koito.db legado e do .bak com fixture descartável
- [x] #7 DEPLOYMENT.md com procedimento passo a passo para recuperar migração interrompida a partir do .bak
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-14T01:22:00Z

---

Lacunas da ISSUE-046 reconciliadas na ISSUE-062: critérios 4 (versão desconhecida sem teste), 5 (injeção de falhas e interrupção por fase), 6 (rollback não exercitado) e 8 (procedimento de recuperação). Os três primeiros critérios desta tarefa já cobrem a injeção de falhas; acrescentados critérios para os demais.

---

author: ebrgr
created: 2026-11-14T01:37:00Z

---

Início: branch `chore/issue-064-testes-de-recuperacao-de-banco` criada a partir de `main`.

---

author: ebrgr
created: 2026-11-14T01:45:00Z

---

Verificação: (1) migração: ENOSPC injetado e queda abrupta de processo filho em cada uma das 8 fases (mais 2 da migração local v2), fixture com 200 escutas pendentes no WAL, cada coluna legada comparada por hash após nova partida; teste de mutação confirma que os testes detectam publicação antecipada e falta de limpeza; (2) importação: falha de disco após 7/20 e encerramento após 12/30 mantêm o que foi gravado e a reimportação completa sem duplicar; arquivo truncado em 4 pontos não grava nada; (3) ensaio de restauração a partir do .bak (teste Go e validate_rollback.py), documentado em backlog/evidencias/recuperacao/ISSUE_064.md; (5) versão desconhecida recusada sem alterar nada (legado v4 e atual v99); (6) rollback para a 0.5 compilada de 6b13514: no mesmo diretório e a partir do .bak; a 0.8 recusa voltar ao diretório depois que a 0.5 altera o koito.db (documentado); (7) DEPLOYMENT.md com procedimento passo a passo, cujos passos têm teste (restauração, sidecars, reconciliação nos dois sentidos). Critério 4: lacunas listadas na evidência aguardam aceite do usuário como limitações. make check verde (10/10 e2e); bootstrap também sob -race. Commits 1938415 e 1b5f494.

---

author: claude
created: 2026-11-14T01:48:00Z

---

Aceite: aprovada pelo usuário em 13/11/2026, incluindo as limitações registradas em backlog/evidencias/recuperacao/ISSUE_064.md (disco cheio real, volumes de rede, interrupção dentro do goose.Up, sobras de queda sem limpeza automática, rollback fora do CI).

---

author: claude
created: 2026-11-14T01:48:00Z

---

Publicação: integrada ao main por fast-forward sem versão nem tag, a pedido do usuário; a entrada do CHANGELOG segue em [Unreleased] para a próxima versão publicada.

---
<!-- COMMENTS:END -->
