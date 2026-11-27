---
id: ISSUE-203
title: >-
  203 — Banco, nível 1 (parte A): remover estruturas sem uso e constantes
  legadas do recibo
status: Aberta
assignee: []
created_date: '2026-11-24T00:51:00Z'
updated_date: '2026-11-27T15:21:00Z'
labels:
  - banco
  - ingestao
milestone: m-11
dependencies: []
references:
  - >-
    backlog/docs/estudos/doc-047 -
    Auditoria-de-uso-e-repetição-dos-dados-do-banco-schema-23.md
  - backlog/evidencias/db-auditoria/colunas.csv
priority: medium
type: chore
ordinal: 216000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: doc-047 (auditoria de uso do banco, seção 4) e doc-046 §5. Remover do esquema final o que não tem escritor, não tem leitor ou só guarda um valor da época da sombra. (1) Tabelas sem escritor no código de produção: ingest_event_details e outbox_details, com as entradas da rotina de retenção (ingest_retention.go) e a coluna integrations.details_retention_days. (2) Constante legada de política: relay_delivery_gate (vale durable em todo banco) e ingest_events.delivery_policy, com os gatilhos trg_relay_policy_one_way e trg_relay_active_policy. (3) Constantes legadas de ingest_events: mode e resolver_variant, e os gatilhos trg_ingest_event_mode_update, trg_ingest_event_shadow_insert, trg_ingest_resolver_variant_immutable, trg_occurrence_receipt_admission, trg_occurrence_link_origin_insert, trg_occurrence_link_origin_update e trg_identity_listen_sync_v9; as consultas de ingestão, saúde e exclusão que filtram esses valores passam a não filtrar. (4) Colunas sem nenhuma referência: ingest_events.uncertainty_ms, client_name, client_version, normalized_snapshot, normalizer_version; listens.review_distinct. FORA DESTA ISSUE: integrations.token_ref, revoked_at e diagnostic_enabled, que ficam por decisão do usuário em 27/11/2026, reservadas às ISSUE-086 a 091, e legacy_relay_binding (ISSUE-209). (5) relay_sent_archive, que o gatilho trg_relay_archive_sent grava e que nenhuma consulta de produção lê: remover a tabela, o gatilho e as exclusões em relay_outbox.go e sqlite.go. Efeito esperado sobre o schema 25: de 45 para 41 tabelas e de 61 para 51 gatilhos, e cerca de 23 bytes a menos por recibo (estimativa medida no schema 23, a medir de novo). CAMINHO DE MIGRAÇÃO (decisão do usuário em 27/11/2026): migração própria desta issue, depois consolidada pela ISSUE-195, com uma branch com migração por vez. Como ingest_events tem 21 índices e 20 gatilhos, remover colunas dela exige recriar a tabela. Nível 1 aprovado pelo usuário em 27/11/2026.

**Correção de 27/11/2026 (conferida no schema 25).** `mode` é citado por 14 gatilhos, e não só pelos desta lista. Seis deles estão entre os 9 a remover; os outros oito precisam ser reescritos sem `mode`: `trg_ingest_revision_active_insert`, `trg_outbox_active_insert`, `trg_ingest_link_insert`, `trg_ingest_link_update`, `trg_ingest_resolution_version`, `trg_recording_move_revisions`, `trg_relay_admission_generation` e `trg_enrichment_enqueue`. Três índices também o citam: `idx_ingest_active_queue`, `idx_ingest_resolver_queue` e `idx_ingest_health_review`. O enriquecimento depende disso em dois lugares: o gatilho `trg_enrichment_enqueue` (migração 025) e `enqueueListenBrainzEnrichment` (`internal/db/sqlite/enrichment_listenbrainz.go`) filtram `mode='active'`. No Go de produção, uma busca por texto aponta uso de `mode` em cerca de 11 arquivos, `delivery_policy` em 7 e `resolver_variant` em 5. Antes de remover cada um dos 9 gatilhos, conferir se ele guarda alguma regra além da época da sombra.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Tabelas ingest_event_details, outbox_details, relay_delivery_gate e relay_sent_archive fora do esquema final, com o gatilho trg_relay_archive_sent e as exclusões correspondentes no Go e na rotina de retenção
- [ ] #2 Colunas legadas e sem uso de ingest_events e listens removidas, as consultas que filtravam mode e resolver_variant ajustadas, e os oito gatilhos e três índices restantes que citam mode reescritos sem ele, com o enriquecimento (trg_enrichment_enqueue e enfileiramento do ListenBrainz) testado
- [ ] #3 Os 9 gatilhos de sombra e política removidos; contagem final conferida sobre o schema 25 (41 tabelas e 51 gatilhos, contando relay_sent_archive e seu gatilho)
- [ ] #4 Testes de ingestão, saúde, exclusão e relay verdes; salto v2 → atual com digest e contagens por perfil iguais
- [ ] #5 Migração própria, com número combinado com a das ISSUE-217 e 218 (uma branch com migração por vez), para a ISSUE-195 consolidar
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-27T15:00:00Z

---

27/11/2026: texto corrigido sobre o schema 25. Contagens (45 para 42 tabelas, 61 para 52 gatilhos, 21 índices e 20 gatilhos em ingest_events), migração 026 em vez de 024 e a lista dos oito gatilhos e três índices que citam mode além dos nove a remover, incluindo os do enriquecimento.

---

author: ebrgr
created: 2026-11-27T15:21:00Z

---

Decisão do usuário em 27/11/2026: nível 1 aprovado (A1); token_ref, revoked_at e diagnostic_enabled ficam (A2); relay_sent_archive passa a ser removida aqui (B5); migração própria, consolidada depois pela 195 (A6).

---
<!-- COMMENTS:END -->
