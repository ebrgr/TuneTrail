# [HANDOFF ISSUE-216]

**Feito:** `internal/enrichment/musicbrainz.go` busca identidade por nome, valida score, duração, álbum e créditos, registra declarações datadas e usa somente o orçamento compartilhado; o cache em memória é limitado a dez minutos. `engine/enrichment.go` registra a fonte entre Navidrome e ListenBrainz, indisponível sem orçamento. Os testes simulados cobrem o provedor, a fila e a cadeia SQLite com o ListenBrainz real e transporte simulado. Documentação e changelog foram atualizados. A consulta ISRC foi implementada e testada como função isolada, sem alterar o Snapshot ou o digest.

**Pendente:** Advisor (Opus, high) e depois Reviewer, antes de qualquer pedido de aceite. A ligação do ISRC ao fluxo depende das ISSUE-218/219; o critério 3 permanece aberto. Prioridade e teto de importação pertencem à ISSUE-213. Cache negativo persistente não foi implementado; proposta registrada no comentário da issue.

**Alertas:** a regra da ISSUE-215 foi escolhida na mesma amostra de 35 escutas e seus resultados são ordem de grandeza, não precisão esperada. Não houve consulta real ao MusicBrainz. `make check` passou com 13/13 percursos; race passou nos pacotes solicitados, incluindo o teste de orçamento sem filtro. Mutações que removeram o filtro de duração e aceitaram o primeiro candidato fizeram os testes falharem.

**Advisor:** examinar a regra de aceitação, ambiguidades (inclusive IDs duplicados), uso exclusivo do Budget, escape Lucene e o tratamento de `length` desconhecido: rejeitado quando a escuta tem duração, mas permitido quando ela não tem, ainda exigindo álbum e créditos. A issue permanece Em andamento.
