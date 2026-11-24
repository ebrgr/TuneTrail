---
id: ISSUE-213
title: 213 — Monitorar enriquecimento e reprocessar histórico com limites
status: Aberta
assignee: []
created_date: '2026-11-24T11:43:00Z'
labels:
  - operacao
  - ingestao
  - ui
milestone: m-13
dependencies:
  - ISSUE-055
references:
  - backlog/evidencias/issue-055/ESTUDO-COMPLEMENTAR.md
type: feature
ordinal: 226000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Permitir enriquecer gradualmente o histórico existente com limites configuráveis de requisições por provedor e apresentar o progresso e a situação do enriquecimento na central de monitoramento ligada à central de saúde. Coordenar com o subsistema das ISSUE-174, 175 e 055, sem bloquear a ingestão e preservando o isolamento por perfil.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Processar histórico de forma gradual e retomável, sem gerar trabalho duplicado, respeitando limites configuráveis por provedor.
- [ ] #2 Mostrar progresso, provedor, resultados, falhas e estado de espera na central de monitoramento, com acesso pela central de saúde.
- [ ] #3 Respeitar permissões e isolamento de perfis; indisponibilidade de provedor não impede registrar escutas.
<!-- AC:END -->
