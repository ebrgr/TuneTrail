# Parecer do Reviewer — ISSUE-186 (commit 228481b)

Escopo: `finishOccurrence`, `resolveStandaloneOccurrence`, `createOccurrence` (ramo sem álbum), remoção do flag `allowStandaloneOccurrences`.
Base: leitura do diff e dos arquivos centrais; testes focados disparados por mim (resultado no comentário da tarefa).

## Achados
- [RISCO MÉDIO, não executado] internal/db/sqlite/ingest_occurrence_catalog.go:createOccurrence — o crédito da ocorrência sem álbum agora liga `artist_id` por `(SELECT artist_id FROM artist_aliases WHERE alias=? COLLATE BINARY LIMIT 1)`, sem `ORDER BY` nem teste de unicidade. Se dois artistas compartilham o mesmo alias (nomes iguais com MBIDs diferentes), a escuta é creditada a um deles de forma arbitrária. Antes o `artist_id` ficava nulo. Correção: ligar só quando houver exatamente um artista com o alias; caso contrário deixar nulo (como antes) ou enviar para `pending_identity`; adicionar teste com dois artistas de mesmo alias.
- [RISCO] a ocorrência sem álbum não é reunida com a mesma faixa de um álbum conhecido (contrato da 132: não atribuir a um álbum). Sem MBID de gravação, o ranking por gravação divide a faixa entre a gravação do álbum e a gravação sem álbum. Coerente com o contrato e com o legado, mas é a principal consequência visível para o usuário; um MBID de gravação declarado resolve.
- [RISCO] MBID de gravação declarado pela fonte, sem prova de álbum, cria a gravação (`musicbrainz_id`) com a proveniência não validada. Dois conteúdos distintos que declarem o mesmo MBID errado compartilham a gravação (o título e os créditos exatos ainda separam as ocorrências). Aceitável pelo contrato; registrar.
- [RISCO] recibos antigos em `pending_identity / release_missing` continuam pendentes e sem ação de reprocessamento automático (não há migração que os reencaminhe). Num banco que nunca ligou a ingestão ativa não existem; no corte da ISSUE-172 vale um passo único que os reenfileire ou os descarte (ISSUE-185).
- [OK] reuso de ocorrência sem lançamento por título/alias exato e créditos exatos na mesma ordem; ambiguidade (várias candidatas) continua `pending_identity`; escuta sem título ou artista continua rejeitada; o mesmo fluxo de decisão de duplicata e de relay das escutas aceitas; remoção do flag de fixture simplifica o produto; paridade de contagem com o legado verificada no lote sintético do ensaio (2.125 nos dois).

Sem BUG executado; um risco médio a corrigir ou aceitar. O Advisor (effort high) é obrigatório por tocar a identidade.
