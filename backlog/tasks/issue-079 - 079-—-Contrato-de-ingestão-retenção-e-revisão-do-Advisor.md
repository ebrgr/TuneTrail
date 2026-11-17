---
id: ISSUE-079
title: 079 — Contrato de ingestão, retenção e revisão do Advisor
status: Em andamento
assignee:
  - '@ebrgr'
created_date: '2026-11-14T00:39:00Z'
updated_date: '2026-11-17T15:32:00Z'
labels:
  - ingestao
  - banco
milestone: m-11
dependencies:
  - ISSUE-064
references:
  - >-
    backlog/decisions/decision-008 -
    Identidade-musical-separada-em-gravação-ocorrência-e-evento.md
  - >-
    backlog/decisions/decision-009 -
    Contrato-de-ingestão-e-retenção-de-recebimentos.md
documentation:
  - doc-039
  - doc-040
priority: high
type: spike
ordinal: 79000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: doc-010 §4. Revisão obrigatória do Advisor antes de implementar (AGENTS.md: mensageria/concorrência/criptografia → @Opus effort=high no Claude, @Sol effort=medium no Codex).

Definir: envelope de evento, decisões de deduplicação (Igual | Diferente | Revisar), retenção de `ingest_events` e guarda do payload bruto (decisão do usuário, doc-010 §10 item 3), e se a migração v4 inclui também as tabelas de metadados da ISSUE-055.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Contrato do envelope e da deduplicação documentado em backlog/docs/design
- [x] #2 Decisão do usuário sobre retenção e payload registrada
- [x] #3 Parecer do Advisor registrado na tarefa
- [x] #4 Plano da migração v4 coordenado com a ISSUE-055
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: codex
created: 2026-11-16T17:06:00Z

---

Contrato da ISSUE-132 aceito em 16/11/2026 (doc-039, decision-008). Coordenar envelope de recebimentos, metadados mínimos de reprocessamento, revisão por perfil e recebimentos conflitantes. Retenção, prazo e guarda de payload bruto continuam decisões desta ISSUE-079; o aceite da 132 não os definiu.

---

author: ebrgr
created: 2026-11-17T12:54:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-079` na branch `spike/issue-079-contrato-de-ingestao-retencao-e` criada a partir de `main`.

---

author: advisor
created: 2026-11-17T12:59:00Z

---

Advisor Modo B: APROVADO PARA PLANEJAMENTO COM RESSALVAS. Plano proposto salvo no ai-memory [PLANO ISSUE-079], page_id 01a1e996-a3fd-7c80-bc2b-b1e278d5d6f4, plans/issue-079-advisor-ingestao-retencao.md. Recomendo ledger mínimo durável, snapshot estruturado versionado, revisão por perfil, deduplicação conservadora e outbox transacional; 90 dias configuráveis somente para detalhes encerrados; bruto desativado, captura diagnóstica saneada opt-in por 7 dias; não incluir schema final da 055 na v4, coordenando metadados em versão posterior à identidade da 133. Plano detalha envelope, conflitos, janelas de candidatura, purga/tombstones, concorrência, migrações/restore em cópia real e impactos propostos nas 080/133/134. Perguntas ao usuário: 1) 90 dias para detalhes? Recomendo sim. 2) bruto desativado com diagnóstico opt-in de 7 dias? Recomendo sim. 3) tabelas da 055 fora da v4? Recomendo sim. 4) manter chaves mínimas após exclusão/descarte para impedir reimportação até esquecimento explícito? Recomendo sim. 5) dedup conservadora com tolerâncias versionadas apenas para candidatura? Recomendo sim. Propostas não constituem decisão aceita. Estudo estático; sem alterações de código, testes, documento formal, ACs, status, push, merge, publish ou Done.

---

author: ebrgr
created: 2026-11-17T15:28:00Z

---

Decisões do usuário (2026-11-17) sobre o parecer do Advisor: 1) APROVADO: detalhes de processamento encerrados com retenção configurável, 90 dias por padrão, preservando escutas, pendências e metadados mínimos. 2) APROVADO: payload bruto desativado por padrão, com diagnóstico saneado OPCIONAL (proposta: 7 dias, por perfil e fonte). 3) APROVADO: as tabelas de metadados da ISSUE-055 ficam fora da migração v4, em migração posterior. 4) NÃO APROVADO: não guardar chaves de reimportação (tombstones) após a exclusão de uma escuta; a reimportação é permitida quando o usuário quiser, e a exclusão não pode bloquear uma importação futura. 5) APROVADO: deduplicação conservadora, com ambiguidades indo para revisão e tolerâncias de tempo apenas para selecionar candidatos.

---
<!-- COMMENTS:END -->
