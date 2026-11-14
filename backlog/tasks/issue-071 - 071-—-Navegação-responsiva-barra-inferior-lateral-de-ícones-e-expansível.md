---
id: ISSUE-071
title: '071 — Navegação responsiva: barra inferior, lateral de ícones e expansível'
status: Aberta
assignee: []
created_date: '2026-11-14T00:38:00Z'
updated_date: '2026-11-14T01:12:00Z'
labels:
  - ui
milestone: m-10
dependencies:
  - ISSUE-069
priority: high
type: feature
ordinal: 71000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: doc-010 §3.3. Requer decisão do usuário sobre a barra inferior no celular (doc-010 §10, item 2).

Passos:
1. < 40rem: barra inferior fixa com 4–5 destinos (Início, Rankings, Histórico, Busca, Mais) e `env(safe-area-inset-bottom)`; cabeçalho de perfil compacto.
2. 40–64rem: barra lateral de ícones com rótulo acessível.
3. > 64rem: barra lateral expansível com rótulos, perfil no topo e seletor de período persistente.
4. Funciona sem JavaScript; foco e leitura por leitor de tela verificados.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Decisão do usuário sobre a barra inferior registrada
- [ ] #2 Três padrões de navegação conforme a largura, sem JavaScript
- [ ] #3 Alvos de toque de pelo menos 44×44px
- [ ] #4 Capturas e aceite visual
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-14T01:12:00Z

---

Coordenar com a ISSUE-110: no celular, o modal de tema e idioma (Palette) e o submenu de administração (Settings2) ficam no destino 'Mais' da barra inferior.

---
<!-- COMMENTS:END -->
