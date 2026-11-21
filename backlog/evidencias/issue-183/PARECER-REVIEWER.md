# Parecer do Reviewer — ISSUE-183 (commit 66e843f)

Escopo: migração 014 (schema v14), `archive.go`, `exports.go`, merge/undo/delete por ocorrência, importador de arquivo versionado.
Base: leitura do diff e dos arquivos centrais; testes não reexecutados por mim (o Codex reportou `make check` 13/13 e 39 respostas HTTP equivalentes à v0.55).

## Achados
- [RISCO] internal/db/sqlite/archive.go:mapID — importação de dono (não root) aceita o atalho "namespace local" para `occurrences` sem checar que a ocorrência pertence ao histórico do perfil. Detalhe e exploração no parecer do Advisor.
- [RISCO] db/migrations_sqlite/014_occurrence_archive.sql:trg_recording_move_revisions — a fusão executa, por ocorrência movida, um UPDATE em `ingest_events` e INSERT em `ingest_revisions` dentro da mesma transação: gravação longa (trava o escritor) para gravações com muitas ocorrências. Medir.
- [RISCO] internal/db/sqlite/exports.go — cursor por event_id aleatório pode perder eventos gravados durante a exportação (ver Advisor).
- [RISCO] recording_moves — linhas stage=1 permanecem para sempre após a fusão (necessárias ao desfazer); definir descarte quando as ocorrências deixarem de existir é automático (CASCADE), mas fusões antigas nunca expiram.
- [OK] Fusão não descarta mais eventos do mesmo segundo; exclusão limitada à ocorrência; export v1 preservado com `?version=1`; migração com backup do v13 e guarda de downgrade.

Sem BUG confirmado por execução. O item de segurança do importador deve ser resolvido antes do aceite.
