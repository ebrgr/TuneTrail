---
id: ISSUE-225
title: >-
  225 — Limpar e excluir o conteúdo original dos recibos aceitos após um período
  configurado
status: Aberta
assignee: []
created_date: '2026-11-27T15:21:00Z'
updated_date: '2026-11-27T15:21:00Z'
labels:
  - ingestao
  - banco
  - config
milestone: m-11
dependencies:
  - ISSUE-208
priority: medium
type: feature
ordinal: 238000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Decisão do usuário em 27/11/2026 (B4, no lugar de guardar para sempre): haver a opção de limpar e excluir o conteúdo original dos recibos aceitos depois de um período configurado. Proposta do coordenador, a confirmar: desligada por padrão (nada é apagado até alguém configurar) e as escutas nunca são excluídas, só o conteúdo guardado do recibo. A ISSUE-208 define o que pode ser apagado. Hoje não há retenção em execução: PurgeClosedIngestDetails não tem chamador e só alcança tabelas de detalhes. CUIDADOS conferidos no código: original_snapshot é NOT NULL com CHECK de JSON e é lido pelo resolvedor, pela admissão e repetição, pelo enriquecimento (reserva, fence, projeção de créditos recebidos, índice idx_ingest_received_recording e a evidência de MBID recebido do ListenBrainz), pela exportação com procedência, pelo relay, pela revisão, pela Inbox e por gatilhos da central de saúde. Apagar o conteúdo não pode fazer uma fonte externa sobrescrever um MBID que a escuta trouxe. Se exigir mudança de esquema, combinar com a ISSUE-195.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Período configurável na administração e no conf.toml, desligado por padrão
- [ ] #2 Limpeza e exclusão do conteúdo original após o período, em lotes, retomável e registrada, sem excluir escutas
- [ ] #3 Resolvedor, deduplicação, enriquecimento, exportação, relay e páginas de saúde e revisão com o comportamento definido para recibos limpos, com testes
- [ ] #4 Economia medida em 300 mil escutas e documentação do que deixa de ser reconstituível
<!-- AC:END -->
