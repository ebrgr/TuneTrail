# Parecer do Reviewer — ISSUE-194 (commits 9709211, ae5cad2, eb6fef3, 36d23c7)

**APROVADO.** Nenhum [BUG] executado. Sem mudança de comportamento em schema 21.

## Verificação própria (22/11/2026)
- `make check`: verde, 13/13 jornadas E2E.
- `go test -race -count=1 -timeout 30m`: `internal/db/sqlite` ok (1.207 s), `importer`, `ingest`, `catalog`, `relay`, `admin` ok.
- Busca de remanescentes no código de produção (`finishIngest`, `finalizeActiveReceipt`, `profile_ingest_flags`, `ingestSchemaVersion`, `version < 21`): só resta a string de validação de layout em `ingest_schema.go`. Nenhum teste usa mais o resolvedor v9.
- Escuta ao vivo (Codex, binário final, porta 4127): `playing_now` 200, aparece em `/apis/web/v1/now-playing` e na home; só o `single` entra no histórico. Carga: 109 escutas com o mesmo digest, ACK p95 5,83 ms.

## [OK]
- Removidos os ramos `version < 21` (revisão, saúde, relay), a revisão manual do escritor v9 (`review_distinct`) e a invalidação de recibos v9 ao apagar histórico (a migração 021 recusa recibos v9 abertos, então são inalcançáveis); código sem chamador em `engine/handlers` (`downloadMissingImage`, `safeRatio`).
- Admissão de outbox do relay exige só a geração aprovada no schema 21; fixtures portadas para o resolvedor por ocorrência; referência da interface regenerada para o schema atual.
- `COLUNAS.md` lista colunas e tabelas sem leitura ou escrita no Go (para a ISSUE-195) e as que ainda têm uso.

## [RISCO] baixo
- `staticcheck` e `deadcode` não rodaram (ferramenta e rede); a busca foi por referências, graphify e `go vet` (registrado em LIMPEZA.md).
- Asserções específicas do legado foram retiradas dos testes, listadas e justificadas em LIMPEZA.md.
