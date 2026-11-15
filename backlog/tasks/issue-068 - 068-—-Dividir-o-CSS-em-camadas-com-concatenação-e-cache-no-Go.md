---
id: ISSUE-068
title: 068 — Dividir o CSS em camadas com concatenação e cache no Go
status: Aguardando aceite
assignee:
  - '@ebrgr'
created_date: '2026-11-14T00:38:00Z'
updated_date: '2026-11-15T17:38:00Z'
labels:
  - ui
milestone: m-10
dependencies:
  - ISSUE-067
references:
  - backlog/evidencias/issue-068/relatorio.md
  - backlog/evidencias/issue-068/parecer-reviewer.md
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
- [x] #1 CSS concatenado equivalente ao anterior (salvo espaços e ordem documentada)
- [x] #2 URL do CSS com hash e cache imutável; troca de conteúdo muda a URL
- [x] #3 Nenhum arquivo de CSS acima de 400 linhas
- [x] #4 Capturas sem diferença visual
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-14T00:56:00Z

---

Usar a função de template asset e o cache imutável criados na ISSUE-106; aqui eles passam a valer para o CSS concatenado.

---

author: claude
created: 2026-11-14T15:00:00Z

---

Requisito do parecer do Advisor na ISSUE-061 (item 4): ao concatenar o CSS com hash, reescrever também os url() internos (fontes da ISSUE-070, ícones) com o hash do arquivo referenciado, senão a troca de uma fonte não invalida o cache.

---

author: ebrgr
created: 2026-11-15T17:04:00Z

---

Início: branch `chore/issue-068-dividir-o-css-em-camadas` criada a partir de `main`.

---

author: ebrgr
created: 2026-11-15T17:38:00Z

---

Verificação: 18 partes em @layer concatenadas pelo Go na ordem do manifesto; mesmas 366 regras (verify.py); estilos computados idênticos em 70.710 elementos, 11 páginas × 2 temas × 5 larguras; 126/132 capturas idênticas e 6 com 19–25 px no ícone do pé da barra lateral, que oscila também entre duas capturas da main; URL muda ao alterar uma parte (bundle_test.go); maior arquivo 323 linhas; Reviewer sem bloqueios, 5 ajustes aplicados. Passo 3 (remover regras mortas): nenhuma removida, pois não havia prova de não uso; fica para a ISSUE-069. make check verde, 12/12 E2E.

---
<!-- COMMENTS:END -->
