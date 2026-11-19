---
id: ISSUE-161
title: 161 — Gate de visibilidade das escutas, imagens e cache
status: Em andamento
assignee:
  - '@ebrgr'
created_date: '2026-11-17T17:33:00Z'
updated_date: '2026-11-19T15:14:00Z'
labels:
  - api
  - ui
  - imagens
milestone: m-10
dependencies:
  - ISSUE-159
  - ISSUE-160
references:
  - backlog/decisions/decision-010
  - backlog/evidencias/issue-161/VERIFICACAO.md
documentation:
  - doc-041
priority: high
type: feature
ordinal: 174000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: ISSUE-100 (aceita em 2026-11-17). Contrato normativo: doc-041 e decision-010. Padrão de senhas atual mantido (bcrypt e validação existentes), sem novos limites ou bloqueios por usuário ou origem; o limitador de login compartilhado existente fica como está; perfis existentes e novos nascem públicos (require_login=0); o relay segue ativo mesmo com escutas protegidas (consequência aceita). Escopo: diretório público em / com todos os perfis (nome, slug e avatar, sem contagens nem capas); opção Exigir login para ver as escutas por perfil; gate central antes de buscar ou serializar em páginas, API, busca, fragmentos, gráficos, now-playing, widget, exportação; rota de imagem com contexto de perfil e sem bypass do cache global; 304 e ETag só depois da autorização; cache privado para dados pessoais; alterar o checkbox invalida caches; ingestão e relay inalterados.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Perfil protegido não vaza em páginas, API, busca, widget, imagens, cache nem 304; visitante e usuário sem acesso recebem 401 ou 404 genéricos
- [ ] #2 Diretório mostra todos os perfis; perfil público continua legível sem login; gestão sempre exige login
- [ ] #3 Ingestão e relay de entrada e saída funcionam com perfil protegido (testes)
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: ebrgr
created: 2026-11-19T15:12:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-161` na branch `feat/issue-161-gate-de-visibilidade-das-escutas` criada a partir de `main`.

---

author: codex
created: 2026-11-19T15:14:00Z

---

Verificação preliminar: cache legado de capas permite reutilização por até 30 dias sem consultar a origem. Inventário inicial e decisão sobre o corte registrados em VERIFICACAO.md. Implementação pausada para decisão do usuário.

---
<!-- COMMENTS:END -->
