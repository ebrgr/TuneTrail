---
id: ISSUE-068
title: 068 — Dividir o CSS em camadas com concatenação e cache no Go
status: Aberta
assignee: []
created_date: '2026-11-14T00:38:00Z'
updated_date: '2026-11-14T00:56:00Z'
labels:
  - ui
milestone: m-10
dependencies:
  - ISSUE-067
priority: high
type: chore
ordinal: 68000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: doc-031 fase 3 e doc-010 §3.2.

Passos:
1. Dividir `ui.css` e `admin.css` em `assets/css/` (reset, tokens, base, layout, components/, pages/, utilities) com `@layer` declarando a ordem.
2. O Go concatena os arquivos em ordem na inicialização e serve um único `ui.css` com hash do conteúdo na URL e `Cache-Control: immutable`.
3. Remover regras mortas encontradas no caminho (registrar cada remoção).
4. Estender `check_web_style.py` para `assets/css/`.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 CSS concatenado equivalente ao anterior (salvo espaços e ordem documentada)
- [ ] #2 URL do CSS com hash e cache imutável; troca de conteúdo muda a URL
- [ ] #3 Nenhum arquivo de CSS acima de 400 linhas
- [ ] #4 Capturas sem diferença visual
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-14T00:56:00Z

---

Usar a função de template asset e o cache imutável criados na ISSUE-106; aqui eles passam a valer para o CSS concatenado.

---
<!-- COMMENTS:END -->
