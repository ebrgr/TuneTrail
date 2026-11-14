---
id: ISSUE-073
title: 073 — Listas, histórico e páginas de mídia compactas, com menu de ações
status: Aberta
assignee: []
created_date: '2026-11-14T00:38:00Z'
updated_date: '2026-11-14T01:06:00Z'
labels:
  - ui
milestone: m-10
dependencies:
  - ISSUE-071
references:
  - backlog/evidencias/ui-responsiva/comparacao-koito-react.md
priority: medium
type: feature
ordinal: 73000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: doc-010 §3.4 (Listas e Mídia).

Passos:
1. Linha compacta no celular: capa, título/artista e horário relativo.
2. Ações administrativas em menu `⋯` (`<details>` ou `popover`), no lugar do link "Manage" repetido.
3. Mídia: cabeçalho com capa e cor de destaque extraída no servidor ao gerar o WebP; abas Visão geral · Faixas · Álbuns · Histórico · Gráficos.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Histórico legível em 320px com ações acessíveis por teclado
- [ ] #2 Cor de destaque calculada no servidor, com contraste verificado
- [ ] #3 Abas funcionam sem JavaScript (links/âncoras)
- [ ] #4 Capturas e aceite visual
- [ ] #5 Cabeçalho de mídia próximo ao Koito: capa grande, tipo, título grande com posição no ranking, contagens e fundo em degradê da cor da capa
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-14T01:06:00Z

---

Comparação com o Koito (/track/23): o cabeçalho de mídia deve seguir o original — capa grande (equivalente a 385px no desktop), tipo, título grande com a posição no ranking (#N), contagens e 'ouvindo desde', e fundo em degradê da cor média da capa (calculada no servidor, já prevista no passo 3). A barra de ícones de gestão e a divisão da página de gestão ficam na ISSUE-108.

---
<!-- COMMENTS:END -->
