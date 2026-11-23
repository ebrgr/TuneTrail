# Parecer do Reviewer — ISSUE-172 fase 2 (HEAD a0ca4f7)

**APROVADO com um débito de manutenção.** Nenhum [BUG] executado.

Verificação própria (worktree, 22/11/2026): `make check` verde com 13/13 jornadas E2E; `go test -race -count=1 -timeout 30m` em `internal/db/sqlite` (ok, 1.163 s), `internal/importer`, `internal/ingest`, `internal/catalog`, `internal/relay/...` (ok). Medições do Codex em `FASE2.md` (ACK p95 11,4–11,6 ms, visibilidade máx. 1,07 s, 0% de 503, 15/s; paridade de digest; salto v2 → v21 com 200 escutas; cópia de 2.826 escutas restaurada com digests iguais).

## [OK]
- Migração 021: transacional, recusa com mensagem acionável quando há recibos de sombra sem vínculo ou do resolvedor antigo abertos, remove `profile_ingest_flags`, gatilhos e `idx_listens_legacy_identity`, cria `profile_ingest_review` (padrão 1) e `awaiting_approval` do relay; `Down` aborta (reversão só por backup).
- Relay em rascunho: salvar não entrega nem cria intenção no outbox; aprovar ativa; DNS visto pelo aprovador, IP/allowlist exigidos na aprovação (`relay_grant.go`).
- Importadores: item acima de 10 KiB recusado individualmente e contado; opcionais truncados em UTF-8; aliases do Koito v1 no orçamento; MBIDs declarados vinculados sem sobrescrever MBID diferente.
- Escuta manual: informa "já existe" na colisão e mantém a contagem.

## [RISCO] médio (manutenção, não executado em produção)
- Código morto do legado ainda no repositório, protegido por `version < 21`: `finishIngest`/`finalizeActiveReceipt` (resolvedor v9), `ingest_activation.go` do SQLite (Preview/SetIngestActivation sem rota), ramos de `SaveListen`, `activeIngestMode`/`occurrenceWriteFlag` e consultas a `profile_ingest_flags` (`ingest_review.go:34,162`, `ingest_resolve.go:231`, `ingest_receipts.go:85,98,111`, `relay_management.go:123`), além de `catalog.SubmitListen` sem chamador fora de testes. Em produção (v2 → última) esses ramos são inalcançáveis, mas existem para os testes de atualização que ainda semeiam schema 20 com o escritor antigo. O critério 1 da tarefa fala em remover o escritor legado: o comportamento foi removido, o código não.

## [RISCO] baixo
- Revisar ligado por padrão em todos os perfis (decisão do usuário): repetições no mesmo segundo vindas de fontes diferentes passam a abrir `pending_duplicate`; vale acompanhar o volume do Inbox/Revisão nos primeiros dias.
- Reenvio com ID externo confiável e Revisar ligado abre pendência (caso f da ISSUE-181; decisão de produto ainda marcada como aberta, mantido o padrão).
- A suíte do sqlite com `-race` leva ~19 min (a medição de 10 min do CI é sem `-race`, ~65 s lá); o passo de race do CI só roda um subconjunto.

---

# Reauditoria — limpeza do código legado (commits 9f010b8 a d2350d3)

**APROVADO.** Nenhum [BUG] executado. A limpeza ficou parcial por escolha explícita do Codex e está registrada em `LIMPEZA.md`.

## Verificação própria (22/11/2026)
- `make check`: verde, 13/13 jornadas E2E.
- `go test -race -count=1 -timeout 30m`: `internal/db/sqlite` ok (1.173 s), `internal/importer`, `internal/ingest`, `internal/catalog`, `internal/relay`, `internal/admin`, `internal/webui` ok.
- Escuta ao vivo conferida por mim num binário do commit anterior à limpeza e pelo Codex no binário final: `playing_now` devolve 200, a faixa aparece em `/apis/web/v1/now-playing` e na home sem virar escuta; um `single` entra no histórico. Testes `TestNowPlaying`, `TestStaleNowPlayingLookupDoesNotEraseNewUpdate`, `TestNowPlayingIsScopedAndMissingTrackClearsOnlyItsProfile`, `TestHomeNowPlayingAppearsBeforePeriodSummary` e `TestMediaNowPlayingMatchesEntityAndProfile` passam.
- Carga (Codex, sem rede externa): ACK p95 7,52 ms no lote misto e 45,80 ms no piloto de 900 escutas a 15/s, 0 de 503, digest igual ao do legado.

## [OK]
- Removidos 4.102 linhas fora do backlog: `catalog.SubmitListen`, `ResolveListenTrack`, `Sqlite.SaveListen` e `SaveListenWithResult`; operações sem rota de ativação e conversão (`ingest_activation.go` do SQLite e do pacote `db`); admissão em sombra e despacho ao resolvedor v9; CLI e scripts de replay/ensaio que dependiam dos modos antigos (preservados no commit `d2ea6ba`).
- As asserções de `TestSubmitListen_*` foram portadas para `PublishNowPlaying`; os testes de atualização passaram a semear por SQL; v2 → 21, v20 → 21 e a prova de backup continuam.

## [RISCO] baixo (débito conhecido)
- Ainda existem no código de produção ramos para schema < 21 em `ingestReviewEnabled`, `healthReviews`, `SetIngestReview` (`ingest_review.go`), `insertRelayIntent` (`relay_outbox.go`) e `RelayPolicyProfiles` (`relay_panel.go`), via `ingestSchemaVersion`. Em produção (v2 → última) são inalcançáveis. `finishIngest` e `finalizeActiveReceipt` só são exercitados por fixtures históricas (`historical_resolution_test.go`). Remover exige portar essas fixtures; fica como issue de acompanhamento ou segunda rodada.
- `staticcheck` e `deadcode` não rodaram (sem rede/ferramenta); a busca foi por referências, graphify e `go vet`.
