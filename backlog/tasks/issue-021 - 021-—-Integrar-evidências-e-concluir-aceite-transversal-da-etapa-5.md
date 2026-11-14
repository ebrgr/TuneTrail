---
id: ISSUE-021
title: 021 — Integrar evidências e concluir aceite transversal da etapa 5
status: Done
assignee: []
created_date: '2026-11-12T23:39:00Z'
labels:
  - validacao
milestone: m-1
dependencies: []
type: task
ordinal: 87000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
> Importada de `docs-dev/issues/021-integracao-aceite-validacao-solved.md` — numeração local 021. Original removido em 12/11/2026; consulte o histórico do Git.

Status: **fechada e resolvida por aceite do usuário em 05/11/2026**. Issue local, sem publicação.
Plano: [roadmap 5](<../docs/roadmaps/doc-008 - Roadmap-5-—-comportamento-permissões-fusos-e-isolamento-entre-perfis.md>). Dependências: 016–020 concluídas.

## Trabalho

- Reunir matriz caso→teste→resultado, incluindo reproduções anteriores e estado corrigido; separar executado, não executado, decisão e limitação aceita.
- Rodar go test ./..., race nos pacotes concorrentes/alterados, vet, typecheck/test/build do cliente e três validadores de servidor sobre bases descartáveis novas e migradas de v2 para v3.
- Exercitar jornadas completas em Go e modo React/API, seleção de perfil, autenticação, ação, retorno e reinício, com no-JS onde aplicável.
- Registrar configurações fictícias, seed, relógio/fuso, modo, ator, perfil, esperado/observado e integridade. Screenshots funcionais apenas quando esclarecem erro/fuso/retorno.
- Solicitar revisão final Advisor das correções e evidências; atualizar roadmap, guias e changelog conforme mudanças.
- Encerrar servidores temporários e apresentar resultado concreto ao usuário para aceite; preparar checklist visual da parte6 sem remover React.

## Aceite

Nenhum caso crítico pendente de autorização/isolamento/integridade. Defeitos temporais reproduzidos resolvidos, decisões estatísticas explícitas, limites conhecidos descritos e Advisor aprovado. React permanece padrão/fallback e SQLite consolidado v3 preservado. Encerrar esta issue só após aceite do usuário, sem publicação/push automático.

## Evidências

Registrar procedimentos e resultados em [roadmap 5](../evidencias/roadmap-05/README.md).
Usar apenas bases SQLite v3 descartáveis e credenciais fictícias. O estudo e o
fechamento de 001–015 não significam que esta issue já foi executada.

## Execução — 05/11/2026

[Resultado consolidado e checklist da parte 6](../evidencias/roadmap-05/ISSUE_021.md).
Foi corrigido o cabeçalho JSON de envio ListenBrainz após regressão. A issue foi fechada por aceite do usuário, após revisão final do Advisor aprovada
com ressalvas; pendências não executadas não são classificadas como aprovadas.
<!-- SECTION:DESCRIPTION:END -->
