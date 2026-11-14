---
id: ISSUE-070
title: 070 — Servir as fontes localmente e remover o Google Fonts
status: Aberta
assignee: []
created_date: '2026-11-14T00:38:00Z'
labels:
  - ui
milestone: m-10
dependencies:
  - ISSUE-068
priority: medium
type: enhancement
ordinal: 70000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: doc-031 fase 5 e doc-010 §3.2 — hoje Jost, League Spartan e Inter vêm do Google Fonts.

Passos:
1. Obter as fontes em woff2 com licença compatível (OFL), apenas os pesos usados; registrar licenças.
2. `assets/fonts/` embutido, `@font-face` com `font-display: swap` e subconjunto latino.
3. Remover os `<link>` do Google Fonts de todos os heads; atualizar a política de privacidade/documentação.
4. Remover os `.ttf` antigos não usados de `assets/`.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Nenhuma requisição a domínios externos ao abrir qualquer página
- [ ] #2 Fontes carregam offline e com licença registrada
- [ ] #3 Tamanho total das fontes medido e documentado
- [ ] #4 Capturas sem regressão tipográfica
<!-- AC:END -->
