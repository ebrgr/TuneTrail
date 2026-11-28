# ISSUE-218 — Terceira rodada de correções

27/11/2026. Veja `PARECER-ADVISOR.md`.

## Regra do usuário

**O MBID sempre tem preferência sobre o ISRC.** O ISRC é evidência para descobrir um ID que falta; nunca prevalece sobre um MBID já conhecido (da escuta, do catálogo ou declarado por outra fonte em `Prior`).

## Itens

1. **Resultado por ISRC em conflito é descartado inteiro** (`internal/enrichment/musicbrainz.go`, `Lookup`). Confira a coerência com os MBIDs conhecidos **antes** de declarar qualquer coisa, inclusive a gravação. Há conflito quando:
   - um crédito com o nome de um artista de ID conhecido tem outro ID; ou
   - um artista de ID conhecido não aparece nos créditos da gravação, nem pelo ID nem pelo nome.
   Em conflito, não declare nada vindo do ISRC e siga para a busca por nome. Descarte por conflito não é falha de consulta. A regra de ambiguidade não muda: mais de uma gravação distinta continua `ambiguous`.
   - Com MBID de gravação conhecido, o ISRC continua sem ser consultado; cubra isso com teste se ainda não houver.
   - `TestReviewerISRCCreditsRejectKnownArtistConflict` afirma o comportamento antigo: ajuste-o.
2. **Guarda de prévia da versão 26** (`internal/db/sqlite/bootstrap.go`): exigir `descriptive_metadata` **e** `metadata_retries`.

Nada além disso. Os demais pontos do parecer viraram tarefas separadas.

## Testes: somente os destes dois itens

- Um teste por caso, que falhe antes da correção. Rode apenas os testes dessas alterações, com `go test -run` (`TUNETRAIL_WEB_DIR=$PWD/assets TUNETRAIL_LANGS_DIR=$PWD/langs`). Não rode `make check`, a suíte completa, o detector de corridas, mutações nem medições.

## Limites

- Resumo curto no `HANDOFF.md`, com os testes rodados. Comentários de código em inglês. Commit com o prefixo `ISSUE-218: `. Sem push, merge, publish ou `Done`. Não invoque outro agente nem subagente. Sem credenciais nem dados pessoais.
