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

## Reauditoria após as correções (commit 99c51fa, schema v15)

Escopo: só o que mudou desde 66e843f (`archive.go`, `exports.go`, migração 015, `track.go`). Rodei `go test -race` em `internal/db/sqlite` (Archive, Export, Merge, Occurrence, Recording) e `engine` (183, Export, Merge): OK.

- [OK] archive.go:mapID — perfil dono só reutiliza ocorrência local já ouvida por ele ou com mapeamento próprio; root inalterado; demais tabelas caem em criação. Teste cobre o caso do Advisor (página da ocorrência do outro perfil segue 404).
- [OK] exports.go — teto de `listen_export_order.sequence` fixado na primeira página (AUTOINCREMENT, sem reuso após exclusão); eventos novos entram na exportação seguinte; v1 preservado.
- [OK] track.go — desfazer expira em 30 dias; limpeza em fusão e desfazer; o ramo "nada a desfazer" confirma a limpeza antes de devolver ErrNotFound.
- [RISCO] db/migrations_sqlite/015_archive_export_boundary.sql — nova tabela e gatilho por INSERT em `listens` (uma escrita extra por escuta) e cópia de todos os eventos na migração; custo pequeno, mas proporcional ao histórico. Há backup do v14 e guarda de downgrade.
- [RISCO] A fusão medida (184 ms, 5.001 ocorrências/10.000 recibos) está abaixo do limite de 300 ms; históricos maiores devem ser remedidos.
- Nota: o agente despachado reescreveu `REVIEW-FOLLOWUP.md` como evidência; este parecer permanece o do Reviewer.

Sem BUG. Os itens 1 a 5 do Advisor estão tratados; o parecer do Advisor continua REPROVADO COM RESSALVAS no registro até o usuário aceitar a reauditoria.
