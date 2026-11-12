---
id: ISS-053
title: 053 — Localizar datas, quantidades e unidades
status: Aguardando aceite
assignee: []
created_date: '2026-11-12T23:39:00Z'
labels:
  - i18n
milestone: m-7
dependencies:
  - ISS-050
references:
  - docs-dev/issues/053-i18n-datas-numeros.md
type: feature
ordinal: 53000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
> Importada de [docs-dev/issues/053-i18n-datas-numeros.md](../../docs-dev/issues/053-i18n-datas-numeros.md) — numeração local 053.

**Status:** Implementada — aguardando teste funcional e aceite do usuário · **Depende de:** 050 · [Estudo](../../docs-dev/ROADMAP_03_UI_MULTILINGUE.md)

## Objetivo

Formatar quantidades, datas, meses, tempo relativo e descrições de gráficos de acordo com idioma e locale aprovados.

## Aceite

- [ ] Tratar plural e parâmetros em mensagens completas; verificar singular, plural, zero e traduções em pt-BR/en.
- [ ] Localizar datas, horários, intervalos, números e rótulos acessíveis sem alterar timestamps, UTC, fuso, DST ou primeiro dia da semana existentes.
- [ ] Preservar ISO, dados `data-*`, SVG, URLs e formatos de entrada; garantir compatibilidade sem JavaScript.
- [ ] Testar limites de calendário, tempo relativo e formatos de máquina, inclusive gráficos e retrospectiva.

## Resultado

Implementação e verificações registradas em [comparação e validação](../../docs-dev/comparisons/i18n/ISSUE_053_054.md). Os itens de aceite permanecem abertos até o teste funcional e a aprovação do usuário.
<!-- SECTION:DESCRIPTION:END -->
