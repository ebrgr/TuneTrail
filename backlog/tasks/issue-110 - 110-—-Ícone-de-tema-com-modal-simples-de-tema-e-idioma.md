---
id: ISSUE-110
title: 110 — Ícone de tema com modal simples de tema e idioma
status: Em andamento
assignee:
  - '@ebrgr'
created_date: '2026-11-14T01:12:00Z'
updated_date: '2026-11-15T01:31:00Z'
labels:
  - ui
  - i18n
  - admin
milestone: m-10
dependencies:
  - ISSUE-122
references:
  - backlog/evidencias/issue-110/advisor-inicial.md
  - backlog/evidencias/issue-110/advisor-segunda-barra.md
  - backlog/evidencias/issue-110/reviewer.md
  - backlog/evidencias/issue-110/painel_tema.log
priority: high
type: feature
ordinal: 110000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: pedido do usuário em 13–14/11/2026; escopo revisto em 14/11/2026 depois do parecer do Advisor (backlog/evidencias/issue-110/advisor-inicial.md e advisor-segunda-barra.md). A segunda barra de administração saiu para a ISSUE-123.

1. O penúltimo ícone da barra (Info, só link para o GitHub) vira Palette (Lucide, ISC). Ao clicar, abre um modal só com a grade de temas, o tema personalizado recolhido em <details> no fim e o idioma (POST /language, sem campos novos), sem abas nem menu lateral. A tecla \\ abre esse modal.
2. A troca de tema sai de ui.js para assets/js/theme-picker.js, carregado no head compartilhado em todos os tipos de página, para funcionar também na administração; o <select data-admin-theme> de admin.js sai.
3. Remover o settings-dialog com abas (Aparência, Conta, Sobre, Administração), o #admin-settings da administração e os <noscript> de idioma do rodapé; um <noscript> curto no modal explica que a troca de tema precisa de JavaScript.
4. Enquanto a ISSUE-123 não chega, o último ícone (Settings2) é um link para /admin, com o selo ShieldCheck da ISSUE-109 quando a sessão está ativa. Sobre (versão e GitHub) vai para a ISSUE-123 como último item da segunda barra.
5. Mesma barra nas páginas públicas, na prévia e na administração (shared/nav.html, ISSUE-122); teclado, foco devolvido, rótulos traduzidos; testes (engine/i18n_ui_test.go) e capturas.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Penúltimo ícone é Palette e abre modal só com temas, tema personalizado recolhido e idioma, sem abas
- [x] #2 Modal de tema abre por clique, teclado e pela tecla \; foco devolvido ao ícone ao fechar
- [x] #3 Troca de tema funciona nas páginas públicas, na prévia e na administração (theme-picker.js compartilhado)
- [x] #4 settings-dialog, #admin-settings e ícone Info removidos; último ícone leva a /admin com selo quando há sessão
- [ ] #5 Traduções em inglês e português; testes e capturas desktop e celular com aceite visual
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-15T00:03:00Z

---

Herdado da ISSUE-067 (decisão do usuário em 14/11/2026): unificar a barra lateral nesta tarefa. Hoje existem duas: partial/sidebar (público e prévia, em assets/templates/public/page.html) e admin/sidebar (assets/templates/admin/shell.html). Diferem na busca (diálogo × link), nos ícones (função icon × SVG no template), nos campos do avatar e no diálogo de configurações. Critério a cumprir aqui: barra lateral definida uma única vez, como já foi feito com o head em templates/shared/head.html; verificar com capturas antes/depois e aceite visual. Ver doc-035.

---

author: ebrgr
created: 2026-11-15T00:50:00Z

---

Início: branch `feat/issue-110-barra-lateral-icone-de-tema` criada a partir de `main`.

---

author: claude
created: 2026-11-15T00:53:00Z

---

Parecer do Advisor (Opus, medium): REPROVADO COM RESSALVAS — 4 lacunas corrigíveis (theme-picker.js compartilhado, Conta como link sem formulário, CSS/fallback de popover, teste i18n_ui). Desenho: webassets.Nav + shared/nav.html. Aguardando decisões do usuário.

---

author: claude
created: 2026-11-15T00:57:00Z

---

Decisões do usuário em 14/11/2026 sobre o parecer do Advisor: (1) unificação da barra lateral numa tarefa própria de refatoração, antes da 110; (2) tema personalizado mantido recolhido em <details> no fim do modal de tema; (3) na prévia, o menu de administração mostra Sobre e um aviso de prévia, sem links para /admin; (4) a tecla \ passa a abrir o modal de tema. A 110 fica com: ícone Palette e modal de tema (temas, personalizado recolhido, idioma, sem abas), menu de administração (Administração, Configuração do servidor, Conta, Sobre) com selo ShieldCheck, popover com fallback, theme-picker.js compartilhado e remoção do settings-dialog/#admin-settings.

---

author: claude
created: 2026-11-15T01:12:00Z

---

Nova demanda do usuário: segunda barra de administração no lugar do submenu. Parecer do Advisor: REPROVADO COM RESSALVAS (popover inadequado, CSRF no Sair, tooltips sobre a segunda barra, destinos dos itens); propõe disclosure com aprimoramento progressivo e dividir a tarefa. Aguardando decisões do usuário.

---

author: claude
created: 2026-11-15T01:19:00Z

---

Escopo revisto por decisão do usuário: 110 só com o tema (Palette, modal de tema e idioma, theme-picker.js, remoção dos diálogos antigos; Settings2 vira link para /admin com selo). Segunda barra na ISSUE-123; Sobre será o último item dela; layout em telas largas com as duas formas (empurrar ou sobrepor) e uma opção para escolher.

---
<!-- COMMENTS:END -->
