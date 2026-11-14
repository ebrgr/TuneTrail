---
id: ISSUE-106
title: 106 — Cache dos arquivos estáticos e do botão Voltar
status: Aberta
assignee: []
created_date: '2026-11-14T00:55:00Z'
updated_date: '2026-11-14T15:10:00Z'
labels:
  - ui
milestone: m-10
dependencies:
  - ISSUE-066
references:
  - backlog/evidencias/ui-responsiva/comparacao-koito-react.md
priority: high
type: enhancement
ordinal: 106000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: comparação com o Koito React (evidência em backlog/evidencias/ui-responsiva/comparacao-koito-react.md). Vitória rápida, independente do redesign.

Problema: `/web-assets/*` sai sem `Cache-Control`, `ETag` ou `Last-Modified`, então o navegador baixa `ui.css` (28 KB), `themes.css`, `theme.js` e os demais scripts a cada navegação. As páginas públicas usam `Cache-Control: no-store`, o que as exclui do cache de voltar/avançar (bfcache): voltar de uma busca refaz a página. O Koito serve arquivos com hash e `immutable`.

Passos:
1. Calcular na inicialização o hash do conteúdo de cada arquivo embutido e expor uma função de template `asset` que gera `/web-assets/ui.css?v=<hash>`.
2. Servir `/web-assets/*` com `Cache-Control: public, max-age=31536000, immutable` quando a URL tiver o hash correto, e com `ETag` (o próprio hash) e `no-cache` quando não tiver.
3. `themes.css` gerado pelo Go: `ETag` derivado das paletas.
4. Páginas públicas: trocar `no-store` por `private, no-cache` (revalida sempre, fica fora de caches compartilhados, permite bfcache). Administração, login e formulários com CSRF continuam `no-store`.
5. Testes dos cabeçalhos e da troca de hash quando o conteúdo muda.

A ISSUE-068 depois aplica o mesmo mecanismo ao CSS concatenado.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Todos os arquivos de /web-assets com hash na URL e cache imutável; mudança de conteúdo muda a URL
- [ ] #2 Segunda navegação não baixa CSS nem JS de novo (verificado nas ferramentas do navegador)
- [ ] #3 Páginas públicas com private, no-cache; administração continua no-store
- [ ] #4 Voltar de uma busca restaura a página do bfcache no Chrome e no Firefox
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-14T15:00:00Z

---

Requisito do parecer do Advisor na ISSUE-061 (item 4): o hash na URL vale também para arquivos referenciados por url() no CSS (fontes woff2, ícones), não só para os <link> e <script> dos templates. Com a decisão da 061 (assets servidos do disco), implementar sobre a estrutura da ISSUE-066.

---

author: claude
created: 2026-11-14T15:10:00Z

---

Nova ordem da Fase 1 aprovada pelo usuário em 14/11/2026: os assets passam a ser servidos do disco pela ISSUE-066 e os templates são reformatados pela ISSUE-067, então as tarefas de fluidez vêm depois delas para evitar retrabalho e conflitos. Esta tarefa calcula hashes sobre a estrutura nova da 066.

---
<!-- COMMENTS:END -->
