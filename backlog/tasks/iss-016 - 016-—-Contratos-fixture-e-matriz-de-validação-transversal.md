---
id: ISS-016
title: 016 — Contratos, fixture e matriz de validação transversal
status: Done
assignee: []
created_date: '2026-11-12T23:39:00Z'
labels:
  - validacao
milestone: m-1
dependencies: []
references:
  - docs-dev/issues/016-matriz-validacao-transversal-solved.md
type: task
ordinal: 16000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
> Importada de [docs-dev/issues/016-matriz-validacao-transversal-solved.md](../../docs-dev/issues/016-matriz-validacao-transversal-solved.md) — numeração local 016.

Status: **fechada e resolvida**. Issue local, sem publicação.
Plano: [roadmap 5](../../docs-dev/ROADMAP_05_VALIDACAO.md). Dependências: Nenhuma; abre a etapa 5.

## Trabalho

- Inventariar famílias de documentos/actions/API/ListenBrainz e métodos a partir das rotas reais; referenciar os manifestos Go/TS existentes.
- Definir tabela de ator × operação × perfil × modo com resultado esperado, incluindo autenticação, escopo e status/redirect.
- Criar fixture descartável determinística Alpha/Beta/Empty, catálogo compartilhado/exclusivo, sem escutas, mesmo segundo, aliases e imagens; variante >1000 registros.
- Calcular resultados esperados independentemente dos handlers/serviços sob teste; registrar relógio e fuso do processo.
- Distinguir dados pessoais de catálogo global e campos estatísticos all-time versus período. Catalogar diferenças de parsing HTML/JSON aceitas e decisões pendentes.
- Reutilizar helpers existentes, sem framework genérico, dados pessoais ou migração.

## Aceite

Matriz cobre todas as famílias e aponta teste existente ou lacuna por caso. Fixture reproduzível, resultados por perfil conferidos, integridade e chaves estrangeiras válidas. Casos temporais reproduzidos pelo Advisor identificados na 018. Nenhuma alteração de permissão implícita no planejamento.

## Evidências

Registrar procedimentos e resultados em [roadmap 5](../../docs-dev/comparisons/roadmap-05/README.md).
Usar apenas bases SQLite v3 descartáveis e credenciais fictícias. O estudo e o
fechamento de 001–015 não significam que esta issue já foi executada.

## Entrega — 05/11/2026

[Inventário, matriz e oráculos](../../docs-dev/comparisons/roadmap-05/ISSUE_016_MATRIZ.md):
contratos por família, ator, método, perfil e modo; evidências existentes, lacunas
e decisões pendentes identificadas para 017–021. O Advisor revisou a precedência
de autenticação, os erros do store e os limites da fixture e aprovou o resultado após os ajustes.

A fixture descartável `scripts/roadmap05_fixture.py` cria SQLite v3 novo, perfis
Alpha/Beta/Empty e duas variantes determinísticas. As contagens, fronteira de
exportação, integridade e chaves estrangeiras foram verificadas. Não injeta relógio
ou inicia servidor; os testes de sessão, HTTP e correções temporais são trabalho
das issues seguintes. As reproduções de fuso permanecem vinculadas à 018.

Testes: execução da fixture pequena e grande; `python3 -m py_compile`;
`go test` dos pacotes adminweb, adminui, webquery, webui e SQLite (cache Go).
Dados e chaves da fixture são fictícios e o banco temporário é removido.

## Encerramento — 05/11/2026

Fechada e resolvida por aceite explícito do usuário.
<!-- SECTION:DESCRIPTION:END -->
