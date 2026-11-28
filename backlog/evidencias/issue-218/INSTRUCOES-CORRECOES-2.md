# ISSUE-218 — Segunda rodada de correções

27/11/2026. Veja «Segunda revisão» em `PARECER-REVIEWER.md` e as reproduções em `reproducoes-parecer-2.go.txt`.

## Escopo: somente os dois itens obrigatórios

1. **Divergência não é falha passageira** (`internal/enrichment/subsonic.go`, `lookup`). Só erro de chamada em `getSong`/`getAlbum` é repetível. Dados divergentes (`getSong` ou `getAlbum`) são resposta definitiva: mantêm as declarações já seguras, entram no cache e não gravam `metadata_retries`. Esperado: repetir a escuta de uma faixa com IDs completos e metadado de faixa gravado não gera nova chamada.
2. **Cache do Navidrome e `Prior`** (`internal/enrichment/subsonic.go`, `Lookup`). A chave precisa incluir os campos pendentes calculados, para uma consulta com ID em `Prior` não responder por outra sem ele.

Os dois ajustes pequenos do parecer (guarda de versão e índice) ficam **fora** desta rodada.

## Testes: somente os destes dois itens

- Um teste por item, que falhe antes da correção; ajuste os testes existentes que afirmem o comportamento antigo (por exemplo, o caso `mismatch` de `TestReviewerSubsonicPartialIdentityRetries`).
- Rode **apenas** esses testes, com `go test -run` (`TUNETRAIL_WEB_DIR=$PWD/assets TUNETRAIL_LANGS_DIR=$PWD/langs`). Não rode `make check`, a suíte completa, o detector de corridas, mutações nem medições.

## Limites

- Acrescente um resumo curto ao `HANDOFF.md`, dizendo quais testes rodaram.
- Comentários de código em inglês. Commit com o prefixo `ISSUE-218: `. Sem push, merge, publish ou `Done`. Não invoque outro agente nem subagente. Não grave credenciais ou dados pessoais.
