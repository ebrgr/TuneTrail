---
id: ISSUE-075
title: 075 — View Transitions, Speculation Rules e tooltips dos gráficos
status: Aberta
assignee: []
created_date: '2026-11-14T00:38:00Z'
updated_date: '2026-11-19T00:22:00Z'
labels:
  - ui
milestone: m-10
dependencies:
  - ISSUE-073
  - ISSUE-107
priority: medium
type: feature
ordinal: 75000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: ROADMAP item 8, ISSUE-056 e doc-010 §3.5. Escopo reduzido em 2026-11-18 por decisão do usuário (caminho 2): ficam só as melhorias de baixo risco, sem JavaScript novo e sem trocar o <main> por script. A interceptação de formulários e os fragmentos parciais de rankings, histórico e mídia (antigos passos 1 e 2) foram movidos para a ISSUE-168. A busca instantânea e a edição sem recarregar seguem na ISSUE-107.

Passos:
1. `@view-transition { navigation: auto; }` para suavizar as navegações completas.
2. Speculation Rules para pré-carregar links do mesmo perfil, sem pré-carregar ações com efeito colateral (logout, exclusão, qualquer URL não idempotente) nem rotas administrativas, e respeitando o cache existente da ISSUE-106.
3. Gráficos com detalhe por `<title>` e tooltip CSS, mantendo as tabelas acessíveis.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Todas as jornadas continuam funcionando com JavaScript desligado e em navegadores sem suporte a View Transitions e Speculation Rules
- [ ] #2 Nenhum JavaScript novo além do mínimo necessário (orçamento total abaixo de 25 KB comprimido); Speculation Rules não pré-carregam URLs com efeito colateral nem rotas administrativas
- [ ] #3 Gráficos com detalhe por title e tooltip CSS, tabelas acessíveis mantidas; capturas e aceite visual do usuário
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-14T00:56:00Z

---

Busca com sugestões (passo e critério 3) movida para a ISSUE-107, após a comparação com o Koito React (backlog/evidencias/ui-responsiva/comparacao-koito-react.md). Esta tarefa passa a depender da 107 para reaproveitar o mecanismo de fragmentos.

---

author: claude
created: 2026-11-14T15:10:00Z

---

ISSUE-056 encerrada como estudo; dependência retirada.

---

author: claude
created: 2026-11-19T00:22:00Z

---

Escopo reduzido por decisão do usuário (2026-11-18, caminho 2): interceptação de formulários e fragmentos parciais movidos para a ISSUE-168. Motivo: a troca do main por script traz risco de Voltar, foco e leitor de tela e pouco ganho técnico; View Transitions, Speculation Rules e tooltips CSS entregam a maior parte do efeito percebido sem JavaScript novo. A ISSUE-078 continua dependendo da 075; a 168 não bloqueia a 078.

---
<!-- COMMENTS:END -->
