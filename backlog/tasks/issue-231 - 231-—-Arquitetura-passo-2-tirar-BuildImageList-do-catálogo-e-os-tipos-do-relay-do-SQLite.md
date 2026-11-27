---
id: ISSUE-231
title: >-
  231 — Arquitetura, passo 2: tirar BuildImageList do catálogo e os tipos do
  relay do SQLite
status: Aberta
assignee: []
created_date: '2026-11-27T15:45:00Z'
updated_date: '2026-11-27T15:45:00Z'
labels:
  - banco
  - processo
milestone: m-15
dependencies:
  - ISSUE-230
priority: low
type: chore
ordinal: 244000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: pergunta do usuário em 27/11/2026 sobre a separação dos subsistemas e levantamento feito no código da v0.79 (linhas por pacote e grafo de imports). Duas correções pequenas, sem mudança de comportamento. (1) internal/db/sqlite importa internal/catalog em 5 arquivos só para usar BuildImageList (internal/catalog/images.go), que monta as URLs de imagem a partir do perfil no contexto e dos tamanhos do imagecache. Ela não pode ir para internal/models, porque usa db.ProfileID e internal/db já importa models (ciclo): criar um pacote folha para as URLs de imagem, importado por quem precisa. (2) internal/relayoutbox importa internal/db/sqlite por 4 tipos: RelayBatch, RelayItem, ErrRelayConfigInvalid e ErrRelaySourceInvalid. Movê-los para internal/relay (que o sqlite já importa) e fazer o relayoutbox depender de uma interface declarada por ele. RelayItem tem o campo privado due, usado só pelo sqlite: decidir como preservá-lo sem expor estado interno sem necessidade. Remover as exceções correspondentes do teste do passo 1.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 internal/db/sqlite não importa mais internal/catalog, e as URLs de imagem geradas são idênticas, com teste
- [ ] #2 internal/relayoutbox não importa internal/db/sqlite e depende de interface própria
- [ ] #3 Exceções removidas do teste de arquitetura do passo 1
- [ ] #4 Nenhuma mudança de comportamento: make check e testes de relay e de imagens passam, com go test -race nos pacotes tocados
<!-- AC:END -->
