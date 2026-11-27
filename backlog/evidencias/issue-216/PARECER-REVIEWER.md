# Parecer do Reviewer da ISSUE-216

Revisão de 27/11/2026 do diff final `main...HEAD` da branch `feat/issue-216-musicbrainz-como-provedor-de-identidade` (commit `3d6c9f3`), sobre a v0.78. Executada por Claude, que também aplicou as melhorias do Advisor, então **não é uma revisão independente**. O parecer do Advisor está em `PARECER-ADVISOR.md` e não foi refeito.

**Resultado: nenhum item obrigatório. Recomendo corrigir os riscos 1 e 2 (baratos) antes do aceite.**

## Achados

1. `[RISCO]` `internal/enrichment/musicbrainz.go:56-61`: `Instance()` devolve o endpoint cru, e `NewMusicBrainz` não valida a URL. O Subsonic valida (`subsonic.go:47` rejeita usuário embutido, consulta e fragmento) e usa um hash como instância justamente para não guardar credencial. O valor do MusicBrainz é gravado em `metadata_values.instance` (`enrichment.go:369-371` e `:404`) e nas linhas do job. Um espelho próprio com credencial na URL (`https://usuario:senha@host`), configurado em `KOITO_MUSICBRAINZ_URL`, viraria texto no banco e nos backups. A exportação já ofusca a instância (`export_enrichment.go:77`), então o vazamento é só no banco. Correção: validar como o Subsonic (http ou https, sem usuário, sem consulta nem fragmento) e usar como instância um hash do endpoint ou só `esquema://host`.
2. `[RISCO]` `musicbrainz.go:150` e `:173`: os créditos de artista são comparados com `equalMetadata` (`EqualFold` com `TrimSpace`), e o álbum com `normalizedMetadata` (espaços colapsados). A regra medida na 215 usava a mesma normalização nos dois. Um crédito com espaço duplo não casa. Correção: usar `normalizedMetadata` nos créditos e no conflito de artista conhecido.
3. `[RISCO]` `musicbrainz.go:147` e `internal/ingest` (`ArtistCreditNames`): sem `artist_names`, o crédito é a string inteira («A & B»), enquanto o MusicBrainz devolve os créditos separados; uma escuta de vários artistas sem essa lista nunca é resolvida. É perda de cobertura, não de correção, e não sei se a amostra da 215 tinha esse caso. Aceitar e registrar na issue, ou tratar os créditos compostos.
4. `[RISCO]` `internal/db/sqlite/enrichment_chain_test.go`, `TestMusicBrainzIdentityChain`: o recibo do teste não tem duração (`chainReceipt` não define `DurationMS`), então a regra de duração não é exercida dentro da cadeia, e o teste não confere o estado do job nem a instância gravada. A regra de duração está coberta nos testes unitários. Correção: afirmar também `jobState` e a instância.
5. `[RISCO, já registrado]` `musicbrainz.go:224`: a consulta inclui `release:"..."` e a 215 mediu sem ela, então a cobertura medida não vale diretamente (o Advisor confirmou). Registrado no `HANDOFF.md`; repetir as 35 consultas quando houver token.
6. `[RISCO, aceito]` `musicbrainz.go:283`: `lookupISRC` não tem chamador em produção. É o que a issue pede (critério 3 aberto até as ISSUE-218 e 219); remover ou ligar quando essas chegarem.

## Verificado

- **Lógica:** regra de aceitação, ambiguidade por truncamento e por duplicidade, conflito com MBID conhecido e uso do `Prior` conferem com o contrato e com o parecer do Advisor. Todas as chamadas passam por `Budget.Do`, com redirecionamento bloqueado e corpo limitado. `uuid.MustParse` só roda depois de `validUUID`.
- **Testes:** as seis regras novas foram quebradas de propósito e o teste correspondente falhou em todas. O teste de caixa foi corrigido porque o UUID só tinha dígitos. `go test -race` limpo em `internal/enrichment`, `internal/mbz` e `internal/db/sqlite`; `make check` com 13 de 13 percursos, depois das melhorias.
- **Projeto:** comentários em inglês nas funções novas; CHANGELOG em inglês e no formato do arquivo; a variável `TUNETRAIL_DISABLE_MUSICBRAINZ` citada na documentação tem alias no código (`env_alias.go:37`); docs descrevem só artista, faixa e álbum indo ao MusicBrainz, o que bate com o código; escopo respeitado (sem ampliar o orçamento nem mudar o Snapshot).

## Não verificado

Comportamento do servidor MusicBrainz real (limite de URL, lista de releases completa, frase só de pontuação); carga com a fila compartilhada cheia, que é da ISSUE-213.
