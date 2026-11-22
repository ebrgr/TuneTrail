---
id: ISSUE-182
title: >-
  182 — Teto vitalício de tentativas e reprocessamento em massa das falhas do
  resolvedor
status: Em andamento
assignee:
  - '@ebrgr'
created_date: '2026-11-21T21:57:00Z'
updated_date: '2026-11-22T01:24:00Z'
labels:
  - ingestao
milestone: m-11
dependencies:
  - ISSUE-173
references:
  - backlog/evidencias/issue-182/VERIFICACAO.md
type: feature
ordinal: 195000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Follow-up do parecer do Advisor da ISSUE-173. (1) Recibo que sempre excede a lease de 120 s nunca estaciona, pois a lease expirada retoma a tentativa sem gastar orçamento: adicionar teto vitalício de claims (ex.: 5x o orçamento) ou contador separado de recuperações de lease. (2) A janela automática de 15 min (1+2+4+8) estaciona a fila inteira numa queda longa de MusicBrainz/rede e a central de saúde só reprocessa um a um (25 listados): oferecer ação 'reprocessar todos' por perfil (dono e root, POST+CSRF, idempotente) e avaliar não gastar orçamento em falha transitória de dependência externa. Menores: trilha de auditoria do reprocessamento (quem/quando), corrigir o comentário de warnParkedIngest. Parecer: backlog/evidencias/issue-173/PARECER-ADVISOR.md.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Recibo cuja resolução sempre excede a lease é estacionado ao atingir o teto vitalício, com motivo registrado
- [x] #2 A central de saúde oferece reprocessar todos os recibos estacionados do perfil, para dono e root, via POST com CSRF e de forma idempotente
- [x] #3 O reprocessamento registra quem e quando; o comentário de warnParkedIngest descreve a janela real de 1 minuto
- [x] #4 Testes cobrem o teto vitalício, o reprocessamento em massa e o isolamento entre perfis; make check passa
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: ebrgr
created: 2026-11-22T01:06:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-182` na branch `feat/issue-182-teto-vitalicio-de-tentativas-e` criada a partir de `main`.

---

author: Codex
created: 2026-11-22T01:11:00Z

---

Verificação: teto de 25 claims por ciclo, ação em massa por perfil e auditoria unitária e em massa implementados. make check passou (13 E2E), assim como os testes direcionados com -race. Referência ISSUE-075 regenerada. Mediana de 116,20 ms para 10 mil estacionados em SQLite local.

---

author: claude
created: 2026-11-22T01:16:00Z

---

Reviewer: sem BUG; 3 RISCOs de operação (reprocessar todos numa única transação sob o escritor; reprocessamento unitário sem releitura de versões; migração reinicia o orçamento dos recibos abertos). Testes sqlite -race, engine, adminui e webui OK. Ver backlog/evidencias/issue-182/PARECER-REVIEWER.md. Advisor (effort high) recomendado por lease e concorrência; pendente a pedido do usuário.

---

author: advisor
created: 2026-11-22T01:24:00Z

---

Advisor (effort high): APROVADO sem falhas críticas. Melhorias: prazo por tentativa abaixo da lease (o teto só dispara se a tentativa voltar), orçamento menor ao reprocessar lease_exhausted, commit por lote no reprocessar todos, histórico de reprocessamentos. Ver backlog/evidencias/issue-182/PARECER-ADVISOR.md.

---
<!-- COMMENTS:END -->
