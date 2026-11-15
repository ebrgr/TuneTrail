---
id: ISSUE-110
title: '110 — Barra lateral: ícone de tema com modal simples e menu de administração'
status: Em andamento
assignee:
  - '@ebrgr'
created_date: '2026-11-14T01:12:00Z'
updated_date: '2026-11-15T00:57:00Z'
labels:
  - ui
  - i18n
  - admin
milestone: m-10
dependencies:
  - ISSUE-109
references:
  - backlog/evidencias/issue-110/advisor-inicial.md
priority: high
type: feature
ordinal: 110000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: pedido do usuário em 13/11/2026.

Situação atual (`sidebar` em `live.html`/`page.html`, `settings-dialog` em `dialogs.html`): o penúltimo ícone (Info, "About") é só um link para o GitHub; o último (Settings2) abre um modal com menu lateral de abas — Aparência (tema, tema personalizado e idioma), Conta, Sobre e Administração. O modal depende de JavaScript para abrir e para trocar de aba.

Mudança 1 — penúltimo ícone vira tema:
1. Trocar Info por **Palette** (Lucide, ISC). Investigado: Palette comunica os vários temas de cores do TuneTrail; SunMoon sugere só claro/escuro e não cobre os temas Yuu, Pearl e demais; Paintbrush lembra edição de imagem.
2. Ao clicar, abre um modal só com a **escolha de tema e de idioma**, sem menu lateral nem abas: lista de temas e seletor de idioma; o tema personalizado (JSON de 14 cores) fica recolhido em `<details>` no fim, por ser parte da escolha de tema.
3. Abrir sem JavaScript: usar o atributo nativo `popover` com `popovertarget` (abre e fecha sem script); a troca de tema continua no `theme.js` e o idioma continua no formulário `POST /language`, que funciona sem script. Remover a alternativa `<noscript>` do rodapé quando o modal cobrir o caso.

Mudança 2 — último ícone vira administração:
4. Manter **Settings2** (engrenagem), que deixa de ser ambígua depois que o tema ganha ícone próprio. Ao clicar, abre um **submenu** (`popover`, ancorado no ícone) com: Administração (`/admin`), Configuração do servidor (`/admin/settings`), Conta (entrar como administrador ou sair) e Sobre o TuneTrail (versão e link para o GitHub, que saem do antigo ícone Info). Sem suporte a `popover`, o ícone é um link para `/admin`.
5. Com sessão de administrador ativa, o ícone mostra o mesmo selo ShieldCheck da ISSUE-109.
6. Remover o `settings-dialog` com abas e as chaves de tradução que ficarem sem uso.

Comum:
7. Mesmas mudanças na barra da prévia (`page.html`) e nas páginas reais; teclado (Tab, Enter, Esc), foco devolvido ao ícone, `aria-label` e `aria-expanded` traduzidos em `langs/`.
8. Coordenar com a ISSUE-071: no celular, tema e administração ficam no destino "Mais" da barra inferior.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Penúltimo ícone é Palette e abre modal só com tema e idioma, sem menu lateral
- [ ] #2 Modal de tema e submenu abrem sem JavaScript (popover) e por teclado
- [ ] #3 Último ícone abre submenu com Administração, Configuração do servidor, Conta e Sobre
- [ ] #4 Ícone Info removido; informações do projeto acessíveis em Sobre
- [ ] #5 Selo de administrador no ícone de administração quando a sessão está ativa
- [ ] #6 Traduções em inglês e português; capturas desktop e celular com aceite visual
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
<!-- COMMENTS:END -->
