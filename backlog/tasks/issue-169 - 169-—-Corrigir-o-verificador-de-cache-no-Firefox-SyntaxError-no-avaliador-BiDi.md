---
id: ISSUE-169
title: >-
  169 — Corrigir o verificador de cache no Firefox (SyntaxError no avaliador
  BiDi)
status: Aberta
assignee: []
created_date: '2026-11-19T10:51:00Z'
labels:
  - ui
milestone: m-10
dependencies: []
references:
  - backlog/evidencias/ISSUE-075/verificacao.md
priority: low
type: bug
ordinal: 182000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: ISSUE-075 (2026-11-19). O verificador scripts/validate_web_cache.cjs falha na etapa Firefox com 'SyntaxError: missing ) in parenthetical' no avaliador BiDi (linha 129, chamada na linha 264, com duas instruções dentro de uma expressão). Chrome e a rodada HTTPS passam. Efeito: a matriz Chrome/Firefox/HTTPS de make web.cache.check não fecha e a ISSUE-075 ficou com o AC1 sem prova completa em navegador sem suporte. Escopo: corrigir o avaliador (por exemplo, função anônima envolvente ou duas chamadas), manter o contrato de cache da ISSUE-106 e rodar a matriz completa. Sem mudança de comportamento em runtime do aplicativo (script de validação).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 make web.cache.check roda Chrome, Firefox e HTTPS sem SyntaxError e a matriz completa passa
- [ ] #2 Contrato de cache da ISSUE-106 inalterado; nenhuma mudança fora do script de validação
<!-- AC:END -->
