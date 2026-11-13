---
id: ISSUE-022
title: 022 — Preparar matriz, fixture e protocolo de comparação
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
ordinal: 22000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
> Importada de `docs-dev/issues/022-matriz-fixture-comparacao-visual-solved.md` — numeração local 022. Original removido em 12/11/2026; consulte o histórico do Git.

Status: **fechada e resolvida em 05/11/2026**. Issue local; não publicar.
Plano: [roadmap 6](<../docs/roadmaps/doc-009 - Roadmap-6-—-comparação-visual-e-decisão-sobre-React.md>).

## Trabalho

- Construir matriz de rotas, estados, modos, perfil, ator, tema, viewport,
  fuso, JavaScript e evidência esperada, com prioridades e lacunas explícitas.
- Preparar fixture reproduzível/versionada com perfis 0/1/N, vazio, dados
  compartilhados, mídia, nomes longos/Unicode, empates, paginação, imagens
  ausentes e períodos/DST; registrar seed, datas e oráculos independentes.
- Criar cópias v3 equivalentes para Go/React e cópias isoladas por mutação;
  prever casos migrados v2→v3 sem editar versão do esquema manualmente.
- Definir manifestos, portas, navegador, escala, fontes, bloqueio de rede externa,
  regiões e prontidão verificável de hidratação, imagens e fontes. Não usar
  waits fixos como prova de prontidão ou congelar apenas o relógio do navegador.
- Preparar procedimentos de captura/encerramento sem dados ou segredos reais.

## Aceite

Matriz e fixture reproduzíveis, contagens/fusos oraculares verificadas, cópias
íntegras e procedimentos explícitos para capturas e casos não cobertos. Não
confundir fixture criada com comparação executada.

## Resultado

[Fixture, matriz, protocolo e validações](../evidencias/roadmap-06/ISSUE_022.md).
Comparações visuais e jornadas completas permanecem nas issues 023–025.
<!-- SECTION:DESCRIPTION:END -->
