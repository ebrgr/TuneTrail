---
id: ISS-052
title: 052 — Internacionalizar administração
status: Aguardando aceite
assignee: []
created_date: '2026-11-12T23:39:00Z'
labels:
  - i18n
milestone: m-7
dependencies:
  - ISS-050
references:
  - docs-dev/issues/052-i18n-administracao.md
type: feature
ordinal: 52000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
> Importada de [docs-dev/issues/052-i18n-administracao.md](../../docs-dev/issues/052-i18n-administracao.md) — numeração local 052.

**Status:** Implementada — aguardando aceite · **Depende de:** 050 · [Estudo](../../docs-dev/ROADMAP_03_UI_MULTILINGUE.md)

## Objetivo

Traduzir o fluxo administrativo Go com os mesmos catálogos e escolha de idioma da UI pública.

## Aceite

- [x] Cobrir login, perfis, importação/exportação, chaves, relay, configuração, edição, mesclagem e remoção, com erros, avisos e confirmações.
- [x] Cobrir acessibilidade e mensagens dos scripts essenciais; idioma igual em login, páginas e respostas de erro.
- [x] Preservar segurança, CSRF, no-store, permissões, formatos de máquina e valores de usuário; testar sucesso, falha, sem JavaScript e concorrência.
- [x] Não traduzir chaves de ambiente nem dados de catálogo; registrar pendências visuais na issue 054.

## Resultado

Administração, mensagens e rótulos acessíveis usam os catálogos externos em `langs`. O idioma segue a preferência do navegador e a escolha nas configurações. [Validação e comparações](../../docs-dev/comparisons/i18n/ISSUE_052.md). Datas e plurais permanecem na issue 053; a revisão visual ampla permanece na 054.
<!-- SECTION:DESCRIPTION:END -->
