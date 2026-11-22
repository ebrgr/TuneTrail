# Parecer do Reviewer — ISSUE-184 (commit 5553062, schema v19)

Escopo: migração 019, `ingest_inbox.go`, handler/`/admin/health` (filtros `days` e `key`), admissão de recibos com `api_key_id`, `GetProfileByAPIKey`, reaproveitamento em `reviewTrustedReplay`.
Base: leitura do diff e dos arquivos centrais; testes focados disparados por mim (resultado no comentário da tarefa).

## Achados
- [RISCO] db/migrations_sqlite/019_ingest_inbox.sql — a migração reconstrói `api_keys` (DROP + rename, para tornar o ID AUTOINCREMENT e não reusar IDs de chaves excluídas) e faz o backfill agregando todos os recibos. Conferi a tabela nova contra a da migração 003: colunas, UNIQUE, FK de perfil, índices e o gatilho `trg_api_key_profile_immutable` foram preservados. O Codex mediu 7,0 s no startup com 300 mil recibos, com o escritor bloqueado; cresce com o histórico. Aceitável dentro da janela de manutenção de 10 minutos, com o backup do v18.
- [RISCO] gatilhos em `ingest_events` (INSERT, DELETE e UPDATE com `WHEN` filtrado) mantêm `ingest_inbox_daily`: cada recibo passa por uma escrita extra na admissão e duas na transição de estado (decrementa e incrementa). Write amplification moderada no caminho do ACK; o p95 de 200 ms da ISSUE-172 deve ser remedido nessa versão.
- [RISCO] a classificação `inbox_metadata` (coluna gerada virtual sobre o JSON do snapshot) depende dos nomes de campo do `ingest.Snapshot`; conferi os usados (`title`, `artist`, `release`, `recording_mbid`, `mapped_recording_mbid`, `release_mbid`, `mapped_release_mbid`, `artist_mbids`, `mapped_artist_mbids`, `artist_mbid_mappings`, `duration_ms`) e coincidem com as tags JSON. Uma mudança futura de tag desclassificaria os recibos sem erro; vale um teste que ligue a coluna às tags.
- [RISCO] o exemplo de metadados exibe título, artista e cliente do snapshot ao dono do perfil e ao root; campos limitados a 240 caracteres, sem corpo bruto nem chave. Adequado.
- [OK] a chave bruta nunca sai: `IngestionKeyID` é `json:"-"` e só o ID interno vai ao recibo; chave removida mantém contagens por ID; `api_key_id` sem FK; filtro `key` validado como inteiro e a leitura é sempre por `profile_id` autorizado (chave de outro perfil não vaza); janelas de 24 h, 7 d e 30 d por dias UTC completos mais as duas bordas parciais; medianas de 6,3 a 8,3 ms com 300 mil recibos (alvo de 50 ms).

Sem BUG confirmado.
