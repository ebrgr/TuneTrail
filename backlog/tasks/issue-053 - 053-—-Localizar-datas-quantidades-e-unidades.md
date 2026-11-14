---
id: ISSUE-053
title: 053 — Localizar datas, quantidades e unidades
status: Done
assignee: []
created_date: '2026-11-12T23:39:00Z'
updated_date: '2026-11-13T23:39:00Z'
labels:
  - i18n
milestone: m-7
dependencies:
  - ISSUE-050
type: feature
ordinal: 53000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
> Importada de `docs-dev/issues/053-i18n-datas-numeros.md` — numeração local 053. Original removido em 12/11/2026; consulte o histórico do Git.

**Status:** Implementada — aguardando teste funcional e aceite do usuário · **Depende de:** 050 · [Estudo](<../docs/roadmaps/doc-006 - Estudo-—-interface-multilíngue-item-3-do-roadmap.md>)

## Objetivo

Formatar quantidades, datas, meses, tempo relativo e descrições de gráficos de acordo com idioma e locale aprovados.

## Aceite

- [ ] Tratar plural e parâmetros em mensagens completas; verificar singular, plural, zero e traduções em pt-BR/en.
- [ ] Localizar datas, horários, intervalos, números e rótulos acessíveis sem alterar timestamps, UTC, fuso, DST ou primeiro dia da semana existentes.
- [ ] Preservar ISO, dados `data-*`, SVG, URLs e formatos de entrada; garantir compatibilidade sem JavaScript.
- [ ] Testar limites de calendário, tempo relativo e formatos de máquina, inclusive gráficos e retrospectiva.

## Resultado

Implementação e verificações registradas em [comparação e validação](../evidencias/i18n/ISSUE_053_054.md). Os itens de aceite permanecem abertos até o teste funcional e a aprovação do usuário.
<!-- SECTION:DESCRIPTION:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: Codex
created: 2026-11-13T23:39:00Z

---

Aceite: aprovada pelo usuário em 13/11/2026, com instrução explícita para fechar a ISSUE-053. Encerramento baseado nas verificações e limitações já registradas; sem nova execução de testes nesta etapa.

---
<!-- COMMENTS:END -->
