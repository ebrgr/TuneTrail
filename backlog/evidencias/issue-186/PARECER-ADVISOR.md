# Parecer do Advisor — ISSUE-186 (effort high)

**APROVADO** (commits 228481b + ef74ac3).

## Falhas críticas
Nenhuma.

## Verificado
- `ingest_occurrence_catalog.go` (createOccurrence, ramo sem lançamento): `min(artist_id) ... HAVING count(DISTINCT artist_id)=1` devolve NULL com 0 ou 2+ artistas; nunca escolhe um homônimo arbitrário. Crédito mantém o nome observado.
- `occurrenceCreditConflicts`: `COLLATE BINARY` no JOIN alinha a detecção de homônimos com a regra de igualdade exata usada no resto (antes a colação da coluna podia divergir).
- `ensureOccurrenceArtists`: só é usado no ramo com lançamento, onde o conflito já vai para `pending_identity` antes, sob o mesmo lock de escrita; o erro para `matches>1` é defesa em profundidade e inalcançável sem corrida.
- `finishOccurrence`: sem álbum não bloqueia por homônimo (decisão coerente com o contrato 039/decisão 008: não há catálogo de artista a proteger num balde "Lançamento desconhecido").
- Teste com dois artistas de mesmo alias cobre os dois caminhos; sqlite/engine e `make check` (13/13 E2E) passaram.

## Melhorias (sem bloquear)
1. Crédito sem `artist_id` fica órfão para sempre; se a 172 fase 2 ou um passo de catálogo quiser vinculá-lo depois, precisará de rotina de relink. Registrar como follow-up, não como pré-requisito.
2. Recibos antigos `pending_identity/release_missing` não são reenfileirados: tratar no corte da 172 (reenfileirar ou descartar pela 185).
3. Ranking divide a mesma faixa entre gravação do álbum e gravação sem álbum sem MBID: aceito pelo contrato.
