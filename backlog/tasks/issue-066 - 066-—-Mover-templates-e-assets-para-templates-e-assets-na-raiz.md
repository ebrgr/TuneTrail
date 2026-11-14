---
id: ISSUE-066
title: 066 — Mover templates e assets para templates/ e assets/ na raiz
status: Aberta
assignee: []
created_date: '2026-11-14T00:38:00Z'
labels:
  - ui
milestone: m-10
dependencies:
  - ISSUE-065
priority: high
type: chore
ordinal: 66000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: doc-031 fase 1 (opção D: embutido + leitura do disco em desenvolvimento). Depende das decisões da ISSUE-061.

Passos:
1. Criar `templates/{public,admin,partials}` e `assets/{css,js,icons,public,themes,brand}`; mover os arquivos de `internal/webui` e `internal/adminui` sem alterar conteúdo.
2. `web.go` na raiz (pacote `tunetrail`) com `//go:embed` apenas dos subdiretórios de runtime; `internal/webui` e `internal/adminui` passam a usar esse `embed.FS`.
3. `TUNETRAIL_WEB_DIR`: em desenvolvimento, lê do disco e recarrega por requisição.
4. Ajustar Dockerfile (COPY na etapa de build), Makefile (`node --check`), scripts e testes que citam caminhos.
5. Mover imagens do README para `assets/brand/` e atualizar links.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 HTML e CSS servidos idênticos byte a byte à referência
- [ ] #2 Binário continua autossuficiente; imagem Docker builda e sobe
- [ ] #3 `TUNETRAIL_WEB_DIR` permite editar template/CSS sem recompilar
- [ ] #4 URLs públicas `/web-assets/...` inalteradas
<!-- AC:END -->
