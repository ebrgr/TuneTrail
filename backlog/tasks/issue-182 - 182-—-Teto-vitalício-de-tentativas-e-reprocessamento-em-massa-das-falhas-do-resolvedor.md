---
id: ISSUE-182
title: >-
  182 — Teto vitalício de tentativas e reprocessamento em massa das falhas do
  resolvedor
status: Aberta
assignee: []
created_date: '2026-11-21T21:57:00Z'
labels:
  - ingestao
milestone: m-11
dependencies:
  - ISSUE-173
type: feature
ordinal: 195000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Follow-up do parecer do Advisor da ISSUE-173. (1) Recibo que sempre excede a lease de 120 s nunca estaciona, pois a lease expirada retoma a tentativa sem gastar orçamento: adicionar teto vitalício de claims (ex.: 5x o orçamento) ou contador separado de recuperações de lease. (2) A janela automática de 15 min (1+2+4+8) estaciona a fila inteira numa queda longa de MusicBrainz/rede e a central de saúde só reprocessa um a um (25 listados): oferecer ação 'reprocessar todos' por perfil (dono e root, POST+CSRF, idempotente) e avaliar não gastar orçamento em falha transitória de dependência externa. Menores: trilha de auditoria do reprocessamento (quem/quando), corrigir o comentário de warnParkedIngest. Parecer: backlog/evidencias/issue-173/PARECER-ADVISOR.md.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Recibo cuja resolução sempre excede a lease é estacionado ao atingir o teto vitalício, com motivo registrado
- [ ] #2 A central de saúde oferece reprocessar todos os recibos estacionados do perfil, para dono e root, via POST com CSRF e de forma idempotente
- [ ] #3 O reprocessamento registra quem e quando; o comentário de warnParkedIngest descreve a janela real de 1 minuto
- [ ] #4 Testes cobrem o teto vitalício, o reprocessamento em massa e o isolamento entre perfis; make check passa
<!-- AC:END -->
