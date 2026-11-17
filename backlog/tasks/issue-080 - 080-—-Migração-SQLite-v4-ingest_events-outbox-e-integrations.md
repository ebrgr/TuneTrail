---
id: ISSUE-080
title: '080 — Migração SQLite v4: ingest_events, outbox e integrations'
status: Aberta
assignee: []
created_date: '2026-11-14T00:39:00Z'
updated_date: '2026-11-17T16:18:00Z'
labels:
  - banco
  - ingestao
milestone: m-11
dependencies:
  - ISSUE-079
references:
  - backlog/decisions/decision-009
documentation:
  - doc-040
priority: high
type: feature
ordinal: 80000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: doc-010 §4.1. Migração aditiva a partir do v3, incluindo tunetrail.db e o legado koito.db; atualizar limites e validações do bootstrap. Contrato normativo: doc-040 e decision-009 (ISSUE-079, aceitas em 2026-11-17). Escopo da v4 (doc-040 §10): SOMENTE infraestrutura de ingestão: integrations com identidade estável por perfil; ingest_events como ledger único com núcleo separado dos detalhes/snapshots, observações conflitantes e revisão versionada; outbox, índices de fila e de purga, constraints de isolamento e idempotência. NÃO incluir as tabelas de metadados da ISSUE-055 (migração posterior) e NÃO criar tombstones nem chaves de reimportação (a exclusão de uma escuta nunca bloqueia uma importação futura). Enquanto a ISSUE-133 não introduzir o evento estável, a v4 não aponta FK para coluna inexistente nem trata faixa/segundo como event_id: estruturas inativas, referências futuras opcionais, sem enfileirar nem resolver até a 133; o relay atual não é substituído em silêncio. Atualizar schemaVersion, checkDatabase, bootstrap e validadores de layout/triggers; não editar SQL 001–003; rejeitar versões desconhecidas e schema incompleto; retenção configurável de detalhes encerrados (padrão 90 dias, 7 a 3650) e diagnóstico opcional desativado por padrão (7 dias). Backup consistente antes de migrar (SQLite Backup API ou VACUUM INTO, com WAL). Binário antigo recusa schema novo.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Migração aditiva testada a partir de bancos v3 reais e legados
- [ ] #2 Bootstrap aceita v4 e recusa versões desconhecidas
- [ ] #3 Backup antes de migrar e ensaio de restauração
- [ ] #4 Nenhuma escuta descartada
- [ ] #5 A v4 contém só infraestrutura de ingestão: sem tabelas da ISSUE-055, sem tombstones e sem chave de reimportação
- [ ] #6 Núcleo e detalhes do ledger separáveis; purga dos detalhes encerrados após o prazo configurável (padrão 90 dias) sem apagar escuta, pendência nem núcleo vigente
- [ ] #7 Estruturas da v4 inativas até a ISSUE-133 (sem FK para evento inexistente); o relay atual continua funcionando
<!-- AC:END -->
