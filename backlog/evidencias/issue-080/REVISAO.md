APROVADO

Revisão direta do diff da ISSUE-080, sem delegação, em 17/11/2026. Plano: 01a1eabc-2ce1-7b03-a43a-859056afb029. Escopo: 004 SQL, bootstrap/backup/validação/retenção SQLite, testes e scripts de migração, tarefa e evidências.

[OK] db/migrations_sqlite/004_ingest_infrastructure.sql:1 — migração aditiva transacional, sem alterar listens/001–003; isolamento composto, chaves parciais completas, snapshots imutáveis, guardas de ativação e Down bloqueado.
[OK] internal/db/sqlite/bootstrap.go:119 — limite v4 e validação completa dos layouts; v3 canônica validada antes de expandir, legado preservado, backup sincronizado antes do upgrade.
[OK] internal/db/sqlite/ingest_schema.go:13 — referência DDL canônica independente detecta ausência/alteração de tabelas, constraints, views, índices e triggers.
[OK] internal/db/sqlite/ingest_retention.go:13 — DELETE obtém serialização de escrita, revalida elegibilidade, limita lote e preserva núcleo/pendência/snapshot/evidência/lease.
[OK] scripts/validate_ingest_migration.py:1 — cópia física com sidecars antes de conexão; restauração isolada e resultados apenas agregados.
[OK] internal/db/sqlite/ingest_migration_test.go:1 — rollback real no meio da 004, backup WAL/restore, constraints, retenção e disputa entre conexões; -race verde.
[OK] scripts/e2e.py:33 — jornada v3→v4 incorporada ao gate; relay atual segue coberto sem worker/outbox habilitado.

Impacto estrutural conferido no graphify da worktree: checkDatabase→validateSchemaLayout e bootstrap→backupBeforeUpgrade; nenhum chamador do relay/ingestão atual redirecionado para a nova infraestrutura. Deslop revisado no mesmo diff, sem limpeza fora de escopo. Limites futuros estão no RELATORIO.md; não constituem implementação antecipada das 081/082/133/134.

## Revisão do Claude (Opus 5.5, effort high) — 2026-11-18

Foco pedido pelo usuário: o caminho que importa é Koito **v2** direto para TuneTrail **v4** (o v3 é banco de desenvolvimento).

Bancos v2 REAIS gerados pelo binário do Koito upstream (compilado de `a079fa6`, último commit upstream), em pasta temporária, sem dados pessoais:
- A: v2 criado pelo Koito atual (60 escutas).
- A sem os índices do PR #257 (60 escutas), como um banco criado antes de 30/04/2026.
- B: banco criado pelo Koito de `26fb2ed` (v1) e atualizado pelo Koito atual para v2.

| Caso | v4 original (8bda702) | v4 corrigido |
| --- | --- | --- |
| A | inicia, v4, 60 escutas, koito.db intacto | igual |
| A sem índices | **não inicia** (`required index idx_artist_aliases_artist_id`) | inicia, v4, 60 escutas |
| B | **não inicia** (mesmo erro) | inicia, v4 |

Correção: a migração 004 recria com `CREATE INDEX IF NOT EXISTS` os dois índices que o 001 do Koito ganhou no PR #257 (`idx_artist_aliases_artist_id`, `idx_listens_track_id_listened_at`). Novo teste `TestBootstrapKoitoV2WithoutUpstreamIndexes` (falha sem a correção, passa com ela). `make check` 13/13 e `go test -race ./internal/db/sqlite` verdes.

Achado fora do TuneTrail: a migração 002 do próprio Koito upstream apaga faixas e escutas ao converter um banco v1 (o `PRAGMA foreign_keys = OFF` não tem efeito dentro da transação do goose; o `DROP TABLE tracks` dispara as cascatas). No caso B o koito.db já chega ao TuneTrail sem escutas (40 → 0 no teste). O TuneTrail não piora nem recupera isso.
