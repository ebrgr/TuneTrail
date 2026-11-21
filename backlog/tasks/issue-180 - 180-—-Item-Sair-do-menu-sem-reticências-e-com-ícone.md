---
id: ISSUE-180
title: 180 — Item Sair do menu (+) sem reticências e com ícone
status: Em andamento
assignee:
  - '@ebrgr'
created_date: '2026-11-21T14:04:00Z'
updated_date: '2026-11-21T14:04:00Z'
labels:
  - ui
dependencies: []
type: enhancement
ordinal: 193000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
O item de saída do menu (+) mostrava "Sair…" com reticências e nenhum ícone. Causa do ícone ausente: o LogOut.svg existe em assets/icons, mas não estava registrado em assets/manifest.json, então o helper icon (internal/webassets/nav.go) devolvia vazio. Mudança: registrar o ícone no manifesto, trocar "Sair…" por "Sair" e "Sign out…" por "Sign out" (langs/pt-BR.json, langs/en.json, internal/webassets/access_nav.go) e ajustar o teste engine/navigation_issue163_test.go.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 O item de saída do menu (+) mostra o ícone LogOut no mesmo padrão dos demais itens
- [x] #2 O rótulo não tem reticências em pt-BR nem em inglês
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Verificado: go test de internal/webassets e internal/webui verdes. Não verificado: tela no navegador e make check completo (engine precisa abrir portas, bloqueado no sandbox).
<!-- SECTION:NOTES:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: ebrgr
created: 2026-11-21T14:04:00Z

---

Início: branch `feat/issue-180-item-sair-do-menu-sem` criada a partir de `main`.

---
<!-- COMMENTS:END -->
