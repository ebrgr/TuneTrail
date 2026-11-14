---
id: ISSUE-019
title: 019 — Validar isolamento entre perfis, ingestão e cache
status: Done
assignee: []
created_date: '2026-11-12T23:39:00Z'
labels:
  - validacao
milestone: m-1
dependencies: []
type: task
ordinal: 85000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
> Importada de `docs-dev/issues/019-isolamento-perfis-ingestao-cache-solved.md` — numeração local 019. Original removido em 12/11/2026; consulte o histórico do Git.

Status: **fechada e resolvida**. Issue local, sem publicação.
Plano: [roadmap 5](<../docs/roadmaps/doc-008 - Roadmap-5-—-comportamento-permissões-fusos-e-isolamento-entre-perfis.md>). Dependências: 016; integrar correções temporais da 018.

## Trabalho

- Cruzar home, rankings, histórico, mídia, relações/aliases, busca textual/id, atividade, interesse, summary, export e tocando agora nos perfis Alpha/Beta/Empty.
- Usar item exclusivo de B na rota de A, IDs misturados e filtros conflitantes; comparar resultado pessoal, status e banco, não só HTML.
- Exercitar legado com zero/um/vários perfis, slug renomeado/excluído, slug reservado, duas abas e retorno após exclusão.
- Validar contexto db.WithProfile/WithoutProfile e exceções globais administrativas; falta de perfil não deve ampliar leitura pública silenciosamente.
- Ingestão com chaves A/B no mesmo timestamp, conteúdo tentando reatribuir perfil, duplicata pessoal, revogação e retomada após reinício.
- Atualizar e invalidar now-playing em um perfil sem afetar outro, inclusive faixa excluída/merge/perfil removido; não misturar chave de cache com seleção do navegador.
- Importação respeita destino explícito e export contém somente o perfil; relay usa URL/token do mesmo perfil, com servidores locais separados.
- Distinguir perfil ausente e falha de store, incluindo o 404 genérico atual em ProfileScope; registrar correção/regressão se necessário.

## Aceite

Nenhuma escuta pessoal atravessa contexto; mesmo segundo entre pessoas permanece e duplicata pessoal segue contrato. Catálogo global é tratado como compartilhamento deliberado e busca global exige admin. Alterar aba/slug não troca destino de formulário aberto. Relatórios incluem totais e efeitos por perfil, integridade e ausência de segredo nas respostas públicas.

## Evidências

Registrar procedimentos e resultados em [roadmap 5](../evidencias/roadmap-05/README.md).
Usar apenas bases SQLite v3 descartáveis e credenciais fictícias. O estudo e o
fechamento de 001–015 não significam que esta issue já foi executada.

## Entrega — 05/11/2026

A matriz de perfis, correções de erros, cache e importação, resultados e limites
estão no [relatório da 019](../evidencias/roadmap-05/ISSUE_019.md). Testes Go,
race, jornadas reais e revisão Advisor passaram. Aceite explícito do usuário recebido em 05/11/2026.

## Encerramento — 05/11/2026

Fechada e resolvida por aceite explícito do usuário.
<!-- SECTION:DESCRIPTION:END -->
