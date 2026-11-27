---
id: ISSUE-231
title: >-
  231 — Arquitetura, passo 2: tirar BuildImageList do catálogo e os tipos do
  relay do SQLite
status: Done
assignee:
  - '@ebrgr'
created_date: '2026-11-27T15:45:00Z'
updated_date: '2026-11-27T21:06:00Z'
labels:
  - banco
  - processo
milestone: m-15
dependencies:
  - ISSUE-230
references:
  - backlog/evidencias/issue-231/VERIFICACAO.md
  - backlog/evidencias/issue-231/PARECER-REVIEWER.md
  - backlog/evidencias/issue-231/PARECER-ADVISOR.md
priority: low
type: chore
ordinal: 244000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: pergunta do usuário em 27/11/2026 sobre a separação dos subsistemas e levantamento feito no código da v0.79 (linhas por pacote e grafo de imports). Três correções pequenas, sem mudança de comportamento. (1) internal/db/sqlite importa internal/catalog em 5 arquivos só para usar BuildImageList (internal/catalog/images.go), que monta as URLs de imagem a partir do perfil no contexto e dos tamanhos do imagecache. Ela não pode ir para internal/models, porque usa db.ProfileID e internal/db já importa models (ciclo): criar um pacote folha para as URLs de imagem, importado por quem precisa. (2) internal/relayoutbox importa internal/db/sqlite por 4 tipos: RelayBatch, RelayItem, ErrRelayConfigInvalid e ErrRelaySourceInvalid. Movê-los para internal/relay (que o sqlite já importa) e fazer o relayoutbox depender de uma interface declarada por ele. RelayItem tem o campo privado due, usado só pelo sqlite: decidir como preservá-lo sem expor estado interno sem necessidade. (3) internal/webui, internal/summary e internal/relayoutbox importam database/sql só para comparar com sql.ErrNoRows (webui/handlers.go, webui/rewind.go, summary/timeframe.go e relayoutbox/worker.go): passar a comparar com db.ErrNotFound, que já existe em internal/db/errors.go, fazendo o sqlite devolvê-lo nos métodos consumidos por esses pontos e conferindo os outros chamadores dos mesmos métodos para não mudar o comportamento. Acrescentado em 27/11/2026, por sugestão aceita pelo usuário no aceite da ISSUE-230. Remover as exceções correspondentes do teste do passo 1.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 internal/db/sqlite não importa mais internal/catalog, e as URLs de imagem geradas são idênticas, com teste
- [x] #2 internal/relayoutbox não importa internal/db/sqlite e depende de interface própria
- [x] #3 Exceções removidas do teste de arquitetura do passo 1
- [x] #4 Nenhuma mudança de comportamento: make check e testes de relay e de imagens passam, com go test -race nos pacotes tocados
- [x] #5 internal/webui, internal/summary e internal/relayoutbox não importam database/sql: linha ausente é reconhecida por db.ErrNotFound, com teste
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-27T20:09:00Z

---

Escopo ampliado em 27/11/2026: o usuário aceitou a sugestão de incluir a troca de sql.ErrNoRows nos três pacotes (achado da ISSUE-230).

---

author: ebrgr
created: 2026-11-27T20:11:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-231` na branch `chore/issue-231-arquitetura-passo-2-tirar-buildimagelist` criada a partir de `main`.

---

author: ebrgr
created: 2026-11-27T20:53:00Z

---

Verificação: make check completo; URLs de imagem idênticas às da função original; go test -race nos pacotes tocados; mutações em backlog/evidencias/issue-231/VERIFICACAO.md; exceções de arquitetura de 13 para 8; parecer do Reviewer sem item obrigatório, com recomendação de Advisor por tocar a fila do relay.

---

author: claude
created: 2026-11-27T20:53:00Z

---

Parecer do Reviewer (mesmo agente, sem delegação): APROVADO sem item obrigatório. Advisor recomendado ao usuário por tocar a fila do relay; não acionado.

---

author: advisor
created: 2026-11-27T21:02:00Z

---

Advisor (Opus, effort high, somente leitura): APROVADO, sem falha crítica. Quatro melhorias opcionais em backlog/evidencias/issue-231/PARECER-ADVISOR.md; a nº 1 (registro do teste de ordem) foi corrigida em VERIFICACAO.md; as nº 2 e 3 aguardam decisão do usuário.

---

author: claude
created: 2026-11-27T21:04:00Z

---

Pedido do usuário em 27/11/2026: «arrume a 2 e 3 e depois aceito». Reaberta para aplicar as melhorias 2 e 3 do Advisor.

---

author: ebrgr
created: 2026-11-27T21:06:00Z

---

Verificação: melhorias 2 e 3 do Advisor aplicadas; make check completo; regra leaf-stays-narrow com quatro mutações detectadas e teste de arquitetura com -race; as demais mudanças são comentários.

---

author: claude
created: 2026-11-27T21:06:00Z

---

Aceite do usuário em 27/11/2026: «aceito e publique após e me mostre após as proximas tarefas».

---
<!-- COMMENTS:END -->
