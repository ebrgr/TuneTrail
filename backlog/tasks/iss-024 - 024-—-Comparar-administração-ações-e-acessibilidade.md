---
id: ISS-024
title: 024 — Comparar administração, ações e acessibilidade
status: Done
assignee: []
created_date: '2026-11-12T23:39:00Z'
labels:
  - ui
  - validacao
milestone: m-2
dependencies: []
references:
  - docs-dev/issues/024-comparacao-admin-acessibilidade-solved.md
type: task
ordinal: 24000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
> Importada de [docs-dev/issues/024-comparacao-admin-acessibilidade-solved.md](../../docs-dev/issues/024-comparacao-admin-acessibilidade-solved.md) — numeração local 024.

Status: **fechada e resolvida em 05/11/2026, com limitações registradas**. Issue local, sem publicação.
Plano: [roadmap 6](../../docs-dev/ROADMAP_06_COMPARACAO_VISUAL.md). Depende da 022.

## Trabalho

- Exercitar login real pela UI, conta, perfis, avatar, chaves sem exposição,
  relay local, import/export, edição, imagem, escutas, merge e exclusão em
  cópias descartáveis; incluir erros, confirmação, cancelamento e retorno.
- Verificar sessão expirada, duas abas, planos obsoletos, repetição, efeitos
  sobre cada perfil e distinção de escuta pessoal versus catálogo global.
- Comparar React hidratado e Go nos estados relevantes; registrar pares e regiões
  de desktop/celular, temas e estados abaixo da dobra, com dados fictícios.
- Avaliar acessibilidade pública e administrativa: teclado, foco/retorno de
  foco, Escape, rótulos, feedback, zoom, contraste, tabelas e no-JS essencial.
  Automações complementam inspeção manual e não certificam conformidade integral.

## Aceite

Ações essenciais e retorno funcionam com escopo correto, sem segredo nas
capturas, sem controle essencial inacessível e com evidências e exceções claras.

## Resultado

[Correções, jornadas, capturas e limitações](../../docs-dev/comparisons/roadmap-06/ISSUE_024.md).
O aceite visual final e a auditoria ampla seguem para a 026.
<!-- SECTION:DESCRIPTION:END -->
