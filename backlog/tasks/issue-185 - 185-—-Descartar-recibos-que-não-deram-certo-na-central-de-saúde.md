---
id: ISSUE-185
title: 185 — Descartar recibos que não deram certo na central de saúde
status: Em andamento
assignee:
  - '@ebrgr'
created_date: '2026-11-22T15:05:00Z'
updated_date: '2026-11-22T16:07:00Z'
labels:
  - ingestao
  - ui
milestone: m-11
dependencies:
  - ISSUE-182
references:
  - backlog/evidencias/issue-185/VERIFICACAO.md
  - backlog/evidencias/issue-185/CORRECOES.md
type: feature
ordinal: 198000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Pedido do usuário em 22/11/2026. Hoje a central de saúde reprocessa recibos estacionados (ISSUE-173/182) e decide pendências de duplicata (ISSUE-181), mas não oferece Descartar. Estado do código (lido em 22/11/2026): existe DiscardIngestPending (internal/db/sqlite/ingest_delete.go), que remove um recibo pendente ou em retry (state received, retryable, pending_identity, pending_duplicate, sem evento, modo active, resolvedor por ocorrência) sem deixar rastro, mas não tem rota nem botão; não existe descarte para recibos estacionados (parked_at) nem inválidos; a exclusão de uma escuta já aceita é feita pela página de escutas (DeleteListen, que remove escuta, recibos e itens abertos do relay). Não há descarte em massa. Esta issue herda a decisão em aberto da ISSUE-083 sobre exclusão de histórico e entregas em trânsito. Escopo proposto: (1) ação Descartar, individual e em massa por perfil, nas seções Falhas permanentes e Pendências de revisão da central (dono do perfil e root, POST + CSRF, com confirmação, transação única, CAS por versão, idempotente, isolamento por perfil); (2) rastro mínimo do descarte (quem, quando, motivo e identificador do recibo, sem o corpo bruto) em vez de remoção sem registro; (3) recibos inválidos também descartáveis; (4) regra para intenções do relay já criadas: um recibo que nunca virou escuta não tem intenção de relay; para qualquer recibo ligado a evento, o descarte é recusado e o usuário é direcionado à exclusão de escuta; (5) o que já foi enviado ao relay (ListenBrainz) não é apagado no destino, e isso deve estar explícito na tela e na documentação. Decisões propostas e aguardando confirmação do usuário: rastro mínimo (recomendado) versus remoção sem rastro; escopo limitado a recibos sem escuta, nunca aceitos. Depende da ISSUE-182; recomendada antes da ISSUE-172, porque depois do corte a fila de recibos é a única via.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 A central de saúde oferece Descartar, individual e em massa, para recibos estacionados, pendentes e inválidos do perfil, para dono e root, via POST com CSRF e confirmação, de forma idempotente
- [x] #2 O descarte grava um rastro mínimo (quem, quando, motivo e identificador) sem o corpo bruto; recibos ligados a uma escuta são recusados com orientação para a exclusão de escuta
- [x] #3 A tela e a documentação informam que o que já foi enviado ao relay não é apagado no destino
- [x] #4 Testes de isolamento entre perfis, repetição, concorrência com o resolvedor e com a revisão da ISSUE-181; make check passa
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: ebrgr
created: 2026-11-22T15:34:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-185` na branch `feat/issue-185-descartar-recibos-que-nao-deram` criada a partir de `main`.

---

author: Codex
created: 2026-11-22T15:45:00Z

---

Verificação: descarte individual e em lotes de até 25 por perfil, com confirmação, CSRF, digest, CAS e auditoria mínima no schema v20. As projeções permanecem consistentes. Testes direcionados com -race e make check com 13/13 E2E passaram. A referência ISSUE-075 foi regenerada e conferida no Orca a 390, 1100 e 1500 px. Mediana de 2,498 s de escrita para 10 mil recibos sintéticos, sem tráfego concorrente. Encaminhada à revisão técnica.

---

author: claude
created: 2026-11-22T15:52:00Z

---

Reviewer: provável BUG não executado: ingest_discards tem PK (profile_id, receipt_id) e o INSERT de rastro não trata conflito; o recibo de revisão de reenvio de ID confiável tem ID determinístico, então descartar, reenviar e descartar de novo viola a PK, dá erro 500 e desfaz o lote (recibo indescartável). RISCOs: lotes de 25 com confirmação a cada lote (10 mil = 400 confirmações), sem retenção do rastro. Testes sqlite -race, engine, handlers, middleware, adminui e webui OK. Ver backlog/evidencias/issue-185/PARECER-REVIEWER.md. Advisor recomendado (exclusão de dados), pendente a pedido do usuário.

---

author: Codex
created: 2026-11-22T15:59:00Z

---

Fixed repeated trusted-replay discards, added one confirmation for the full eligible set with transactions of up to 500 receipts, and set a 90-day audit retention window with bounded cleanup. Ten thousand parked receipts took a median of 2.812 s at the repository. make check passed all 13 E2E scenarios, targeted race tests passed, and Orca verified 390, 1100, and 1500 px. Technical review pending.

---

author: claude
created: 2026-11-22T16:07:00Z

---

Reviewer (reauditoria do commit 6981a53): sem BUG; o bug do descarte repetido foi corrigido (id por descarte e geração no CAS), descarte em massa com confirmação única do conjunto e lotes de 500, retenção de 90 dias. RISCOs: releitura do conjunto restante a cada lote (O(n²/500)); recibo novo durante o descarte provoca conflito e interrompe os lotes seguintes. Testes sqlite -race, engine, handlers, middleware, adminui e webui OK. Ver PARECER-REVIEWER.md. Advisor (exclusão de dados) pendente a pedido do usuário.

---
<!-- COMMENTS:END -->
