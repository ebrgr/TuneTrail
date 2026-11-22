---
id: ISSUE-185
title: 185 — Descartar recibos que não deram certo na central de saúde
status: Aberta
assignee: []
created_date: '2026-11-22T15:05:00Z'
labels:
  - ingestao
  - ui
milestone: m-11
dependencies:
  - ISSUE-182
type: feature
ordinal: 198000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Pedido do usuário em 22/11/2026. Hoje a central de saúde reprocessa recibos estacionados (ISSUE-173/182) e decide pendências de duplicata (ISSUE-181), mas não oferece Descartar. Estado do código (lido em 22/11/2026): existe DiscardIngestPending (internal/db/sqlite/ingest_delete.go), que remove um recibo pendente ou em retry (state received, retryable, pending_identity, pending_duplicate, sem evento, modo active, resolvedor por ocorrência) sem deixar rastro, mas não tem rota nem botão; não existe descarte para recibos estacionados (parked_at) nem inválidos; a exclusão de uma escuta já aceita é feita pela página de escutas (DeleteListen, que remove escuta, recibos e itens abertos do relay). Não há descarte em massa. Esta issue herda a decisão em aberto da ISSUE-083 sobre exclusão de histórico e entregas em trânsito. Escopo proposto: (1) ação Descartar, individual e em massa por perfil, nas seções Falhas permanentes e Pendências de revisão da central (dono do perfil e root, POST + CSRF, com confirmação, transação única, CAS por versão, idempotente, isolamento por perfil); (2) rastro mínimo do descarte (quem, quando, motivo e identificador do recibo, sem o corpo bruto) em vez de remoção sem registro; (3) recibos inválidos também descartáveis; (4) regra para intenções do relay já criadas: um recibo que nunca virou escuta não tem intenção de relay; para qualquer recibo ligado a evento, o descarte é recusado e o usuário é direcionado à exclusão de escuta; (5) o que já foi enviado ao relay (ListenBrainz) não é apagado no destino, e isso deve estar explícito na tela e na documentação. Decisões propostas e aguardando confirmação do usuário: rastro mínimo (recomendado) versus remoção sem rastro; escopo limitado a recibos sem escuta, nunca aceitos. Depende da ISSUE-182; recomendada antes da ISSUE-172, porque depois do corte a fila de recibos é a única via.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 A central de saúde oferece Descartar, individual e em massa, para recibos estacionados, pendentes e inválidos do perfil, para dono e root, via POST com CSRF e confirmação, de forma idempotente
- [ ] #2 O descarte grava um rastro mínimo (quem, quando, motivo e identificador) sem o corpo bruto; recibos ligados a uma escuta são recusados com orientação para a exclusão de escuta
- [ ] #3 A tela e a documentação informam que o que já foi enviado ao relay não é apagado no destino
- [ ] #4 Testes de isolamento entre perfis, repetição, concorrência com o resolvedor e com a revisão da ISSUE-181; make check passa
<!-- AC:END -->
