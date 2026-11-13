---
id: ISSUE-023
title: 023 — Comparar páginas públicas, mídia e retrospectiva
status: Done
assignee: []
created_date: '2026-11-12T23:39:00Z'
labels:
  - ui
  - validacao
milestone: m-2
dependencies: []
references: []
type: task
ordinal: 23000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
> Importada de `docs-dev/issues/023-comparacao-publica-midia-retrospectiva-solved.md` — numeração local 023. Original removido em 12/11/2026; consulte o histórico do Git.

Status: **fechada e resolvida em 05/11/2026, com limitações registradas**. Issue local, sem publicação.
Plano: [roadmap 6](<../docs/roadmaps/doc-009 - Roadmap-6-—-comparação-visual-e-decisão-sobre-React.md>). Depende da 022.

## Trabalho

- Comparar React hidratado e Go por cliques: seleção/troca de perfil, busca,
  home, rankings, filtros, paginação, histórico, mídia, gráficos e retrospectiva;
  incluir acesso direto, recarga, voltar/avançar e duas abas.
- Verificar no-JS as tarefas essenciais Go, inclusive seleção de perfil, busca e
  aparência: sidebar usa dialogs acionados por JS por leitura; reproduzir e
  corrigir ou registrar/decidir o fallback, sem presumir que URL direta basta.
- Confrontar contagens por data e cada intervalo dos gráficos com fixture e
  oráculo independente; testar vazio, capa ausente, nomes longos, fuso e DST.
- Registrar pares desktop/celular, Yuu/Pearl/Black representativos, demais
  paletas por amostra, regiões abaixo da dobra, breakpoints e diferenças.

## Aceite

Jornadas essenciais com escopo correto, paridade funcional avaliada e diferenças
classificadas; screenshots rastreáveis e limitações não executadas explícitas.

## Resultado

[Jornadas, correção, capturas e limitações](../evidencias/roadmap-06/ISSUE_023.md).
A aceitação das diferenças visuais integra a 026.
<!-- SECTION:DESCRIPTION:END -->
