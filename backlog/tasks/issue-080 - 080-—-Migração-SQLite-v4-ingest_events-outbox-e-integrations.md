---
id: ISSUE-080
title: '080 — Migração SQLite v4: ingest_events, outbox e integrations'
status: Done
assignee:
  - '@ebrgr'
created_date: '2026-11-14T00:39:00Z'
updated_date: '2026-11-18T16:35:00Z'
labels:
  - banco
  - ingestao
milestone: m-11
dependencies:
  - ISSUE-079
references:
  - backlog/decisions/decision-009
  - backlog/evidencias/issue-080/RELATORIO.md
  - backlog/evidencias/issue-080/REVISAO.md
  - backlog/evidencias/issue-080/migracoes-agregadas.json
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
- [x] #1 Migração aditiva testada a partir de bancos v3 reais e legados
- [x] #2 Bootstrap aceita v4 e recusa versões desconhecidas
- [x] #3 Backup antes de migrar e ensaio de restauração
- [x] #4 Nenhuma escuta descartada
- [x] #5 A v4 contém só infraestrutura de ingestão: sem tabelas da ISSUE-055, sem tombstones e sem chave de reimportação
- [x] #6 Núcleo e detalhes do ledger separáveis; purga dos detalhes encerrados após o prazo configurável (padrão 90 dias) sem apagar escuta, pendência nem núcleo vigente
- [x] #7 Estruturas da v4 inativas até a ISSUE-133 (sem FK para evento inexistente); o relay atual continua funcionando
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: ebrgr
created: 2026-11-17T18:17:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-080` na branch `feat/issue-080-migracao-sqlite-v4-ingest-events` criada a partir de `main`.

---

author: advisor
created: 2026-11-17T18:19:00Z

---

Advisor Modo B: APROVADO. Plano 01a1eabc-2ce1-7b03-a43a-859056afb029 (plans/issue-080-advisor-v4.md): v4 aditiva com integrations, ledger único ingest_events, detalhes separáveis, variantes, revisões e outbox; FKs compostas por perfil e índices únicos parciais para chaves confiáveis. Guardas impedem resolução/event_id/outbox até a 133; relay atual preservado. Retenção 90 dias (7–3650), sem TTL do núcleo/pendências, diagnóstico off. Validar v3 antes de migrar e v4 completa depois; rejeitar versões desconhecidas. VACUUM INTO com WAL, validação/fsync antes da transação Goose; rollback no meio e restauração testados. Testes de constraints, purga, recuperação, cópias reais/legado, relay e make check. Sem dúvida de política; contratos aceitos prevalecem sobre texto histórico proposed. Sem 055, tombstones, edição 001–003 ou reserva de versões após v4. Riscos: espaço de backup, escritores externos e vínculos futuros da 133. Arquivos: 004 SQL, bootstrap/backup/validação/retenção SQLite, testes/scripts e evidências agregadas. Plano integral no ai-memory.

---

author: codex
created: 2026-11-17T18:29:00Z

---

Implementação: 004 aditiva, bootstrap/backup/validação canônica v3/v4 e purga explícita de detalhes. Testes de constraints, isolamento, rollback SQL no meio da transação, WAL/restore, estados/leases/retencao e disputa entre conexões. Cópias reais/legado: P1 80→80; P1/P2 79/1→79/1; descartável 24296/298→24296/298. Todas as tabelas existentes com digests iguais. Binário v3 da main recusou v4 sem alterar dados. Relay preservado; estruturas bloqueadas até a 133. Sem CHANGELOG por ser migração interna conforme skill. Evidências agregadas anexadas.

---

author: codex
created: 2026-11-17T18:30:00Z

---

Verificação: make check verde (Go vet/test, UI, 13/13 e2e incluindo migration, ingest-migration e relay/shutdown); go test -race ./internal/db/sqlite verde. Migrações de cópias v3 reais/legado e descartável preservaram 80, 79+1 e 24296+298 escutas, contagens por perfil e digests de todas as tabelas existentes. Backup WAL antes de migrar, restore isolado, rollback no meio, reinício, schema incompleto/versões desconhecidas e recusa da v4 pelo binário v3 comprovados. Revisão direta APROVADO em backlog/evidencias/issue-080/REVISAO.md; relatório agregado anexado. Sete ACs verificados. Infraestrutura inativa até a 133, sem substituir relay; sem 055/tombstones. Aguarda aceite do usuário; sem Done, push, merge ou publish.

---

author: claude
created: 2026-11-18T16:33:00Z

---

Revisão do Claude (2026-11-18): o caminho Koito v2 → v4 foi testado com bancos v2 reais do Koito upstream. Defeito encontrado: bancos v2 criados antes do PR #257 do Koito (sem idx_artist_aliases_artist_id) faziam o TuneTrail v4 recusar iniciar. Corrigido na 004 (CREATE INDEX IF NOT EXISTS dos dois índices) com teste de regressão; make check 13/13 e -race verdes. Detalhes em backlog/evidencias/issue-080/REVISAO.md.

---

author: ebrgr
created: 2026-11-18T16:35:00Z

---

Aceite do usuário (2026-11-18) após a revisão com o caminho Koito v2 → v4 testado com bancos reais do Koito upstream e a correção dos índices do PR #257. Publicação com push autorizada pelo usuário.

---
<!-- COMMENTS:END -->
