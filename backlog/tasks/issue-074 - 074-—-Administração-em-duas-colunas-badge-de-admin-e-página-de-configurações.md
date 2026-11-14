---
id: ISSUE-074
title: 074 — Administração em duas colunas, badge de admin e página de configurações
status: Aberta
assignee: []
created_date: '2026-11-14T00:38:00Z'
updated_date: '2026-11-14T01:12:00Z'
labels:
  - admin
  - ui
milestone: m-10
dependencies:
  - ISSUE-071
priority: medium
type: feature
ordinal: 74000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: ROADMAP item 2 (layout da administração e badge) e doc-010 §3.4.

Passos:
1. Desktop: navegação de seções à esquerda e conteúdo à direita; celular: seções em cards.
2. Ações destrutivas em zona de perigo separada; "Sair" com peso visual secundário.
3. Badge **Admin** ao lado do nome sempre que a conta de administração estiver logada.
4. Página própria `/settings` (tema, idioma, fuso, início da semana), funcionando sem JavaScript; hoje ficam no rodapé.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Layout de administração em duas colunas no desktop e cards no celular
- [ ] #2 Configurações em /settings sem depender de JavaScript
- [ ] #3 Zona de perigo separada e confirmada
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-14T01:06:00Z

---

Exclusão, mesclagem e edição de itens de mídia (faixa, álbum, artista) saem da administração geral e passam a ser feitas pela barra de ações da página do item (ISSUE-108). Esta tarefa continua cuidando do layout da administração, do badge e de /settings.

---

author: claude
created: 2026-11-14T01:12:00Z

---

Badge de administrador (critério 2 e passo 3) movido para a ISSUE-109, que cria o selo ShieldCheck no cabeçalho do perfil com detecção da sessão no servidor. O acesso à administração pela barra lateral passa a ser o submenu da ISSUE-110; esta tarefa continua com o layout da administração, a zona de perigo e /settings.

---
<!-- COMMENTS:END -->
