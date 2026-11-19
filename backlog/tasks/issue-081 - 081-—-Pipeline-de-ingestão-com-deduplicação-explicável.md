---
id: ISSUE-081
title: 081 — Pipeline de ingestão com deduplicação explicável
status: Aberta
assignee: []
created_date: '2026-11-14T00:39:00Z'
updated_date: '2026-11-19T16:15:00Z'
labels:
  - ingestao
milestone: m-11
dependencies:
  - ISSUE-080
documentation:
  - doc-039
  - doc-043
priority: high
type: feature
ordinal: 81000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: doc-010 §4.2. `internal/ingest` (normalizar → deduplicar → gravar → enfileirar) e `internal/dedup`. Deduplicação dentro do perfil; casos duvidosos vão para revisão, nunca apagados automaticamente. Revisão obrigatória do Advisor antes de implementar (AGENTS.md: mensageria/concorrência/criptografia → @Opus effort=high no Claude, @Sol effort=medium no Codex).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 ListenBrainz e importações passam pelo pipeline sem mudar o comportamento observável
- [ ] #2 Cada decisão grava motivo em ingest_events
- [ ] #3 Testes de concorrência e de repetições legítimas
- [ ] #4 Parecer do Advisor registrado
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: codex
created: 2026-11-16T17:06:00Z

---

Alinhamento após aceite da ISSUE-132: doc-039 e decision-008 exigem preservar recebimentos ambíguos/conflitantes e confirmar ListenBrainz após persistência durável, com revisão separada da deduplicação. O AC1 sobre comportamento observável deve preservar formato e compatibilidade dos clientes, incluindo esta mudança de semântica aprovada para conflitos; não exigir manter o HTTP 500 atual. Playing_now permanece efêmero.

---

author: advisor
created: 2026-11-19T16:15:00Z

---

Parecer do Advisor (Opus, effort medium, 2026-11-19), estudo completo em doc-043. Recomendações 1 a 5 registradas a pedido do usuário como a direção adotada, a confirmar na execução: (1) dividir em 081a (modo sombra: grava recibo e motivos ao lado do caminho atual, sem mudar nenhuma contagem) e 081b (ativação, depois da 133/134), para medir a taxa real de Revisar no banco verdadeiro antes de mudar números; (2) mesma faixa duas vezes no mesmo segundo pela API, sem ID externo: mandar para revisão (contrato §5), não contar como duas; (3) modo Revisar atrás de flag por perfil, desligada até a 083 existir; (4) escuta sem listened_at: aceitar com a hora de chegada marcada como estimated; (5) bug do mbid_mapping (lbz_submit_listen.go:203) corrigido em tarefa própria, fora da 081 (ISSUE-170). Ordem: 081a antes da 133; 133 → 081b e 134 → 082/083; 085/086 por último; migração da 133 será v7 ou posterior. A confirmar na 081a: se a v4 aceita duplicate_decision same sem event_id (inferência do Advisor). Advisor effort=high continua obrigatório sobre o diff do passo 6 (ativação).

---
<!-- COMMENTS:END -->
