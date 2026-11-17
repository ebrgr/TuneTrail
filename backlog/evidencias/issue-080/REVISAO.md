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
