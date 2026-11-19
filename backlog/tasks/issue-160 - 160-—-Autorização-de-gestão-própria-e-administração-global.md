---
id: ISSUE-160
title: 160 — Autorização de gestão própria e administração global
status: Em andamento
assignee:
  - '@ebrgr'
created_date: '2026-11-17T17:33:00Z'
updated_date: '2026-11-19T00:36:00Z'
labels:
  - admin
  - api
milestone: m-10
dependencies:
  - ISSUE-158
  - ISSUE-159
references:
  - backlog/decisions/decision-010
  - backlog/evidencias/issue-160/AMBIGUIDADE-IMPORTACAO.md
  - backlog/evidencias/issue-160/VERIFICACAO.md
  - backlog/evidencias/issue-160/REVIEWER.md
documentation:
  - doc-041
priority: high
type: feature
ordinal: 173000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: ISSUE-100 (aceita em 2026-11-17). Contrato normativo: doc-041 e decision-010. Padrão de senhas atual mantido (bcrypt e validação existentes), sem novos limites ou bloqueios por usuário ou origem; o limitador de login compartilhado existente fica como está; perfis existentes e novos nascem públicos (require_login=0); o relay segue ativo mesmo com escutas protegidas (consequência aceita). Escopo: o dono maneja somente os seus dados, chaves, importação e exportação; root maneja todos os perfis e o servidor; catálogo compartilhado (capas, aliases, merges) só root; papel lido no servidor a cada requisição; último admin ativo protegido em transação; sem mass assignment de role, status ou dono.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Matriz de permissões do doc-041 coberta por testes (visitante, usuário, root) em HTML, JSON e serviços
- [x] #2 Último admin protegido em transação, com teste concorrente; usuário não se promove
- [x] #3 Catálogo compartilhado exclusivo do root; chave de ingestão não administra nem lê dados privados
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: ebrgr
created: 2026-11-19T00:05:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-160` na branch `feat/issue-160-autorizacao-de-gestao-propria-e` criada a partir de `main`.

---

author: codex
created: 2026-11-19T00:07:00Z

---

Verificação parcial: análise estática identificou ambiguidade entre a importação própria permitida no doc-041 §2 e a criação de capas/aliases compartilhados pelo ImportKoitoFile (internal/importer/koito.go:105,141,184). Execução interrompida conforme instrução explícita, aguardando decisão registrada em backlog/evidencias/issue-160/AMBIGUIDADE-IMPORTACAO.md. Nenhum runtime alterado, nenhum AC marcado; make check e race não executados. Mantido Em andamento.

---

author: codex
created: 2026-11-19T00:20:00Z

---

Decisão do usuário: opção 2. Importação própria pode criar entidades novas com capas e aliases; alterações editoriais de entidades existentes continuam exclusivas do root. Implementação retomada na mesma worktree.

---

author: codex
created: 2026-11-19T00:36:00Z

---

Verificação: política central de gestão aplicada em HTML, JSON e serviços; dono restrito ao próprio perfil, catálogo/servidor/contas exclusivos root; serviços reconsultam sessão e rejeitam API key/role obsoleto; mass assignment negado. Migração v6 protege último admin ativo e serviço de acesso revalida root dentro da transação, com concorrência em duas conexões para excluir/rebaixar/desativar. Opção 2 da importação testada. make check passou (13/13 E2E); race nos pacotes tocados passou com timeout 10m, e casos finais 160 repetidos com race. Graphify AST atualizado nesta worktree, diff/deslop e revisão local APROVADO. ACs comprovados no escopo de gestão da 160; gate de leitura require_login/imagens/cache (161), fluxos de token (162) e navegação/destino (163/157) permanecem fora do escopo. Evidências: VERIFICACAO.md e REVIEWER.md. Mantido Em andamento, sem push/merge/publish/Done.

---
<!-- COMMENTS:END -->
