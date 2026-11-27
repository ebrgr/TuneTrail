# [HANDOFF ISSUE-216]

**Feito:** `internal/enrichment/musicbrainz.go` busca identidade por nome, valida score, duração, álbum e créditos, registra declarações datadas e usa somente o orçamento compartilhado; o cache em memória é limitado a dez minutos. `engine/enrichment.go` registra a fonte entre Navidrome e ListenBrainz, indisponível sem orçamento. Os testes simulados cobrem o provedor, a fila e a cadeia SQLite com o ListenBrainz real e transporte simulado. Documentação e changelog foram atualizados. A consulta ISRC foi implementada e testada como função isolada, sem alterar o Snapshot ou o digest.

**Pendente:** Advisor (Opus, high) e depois Reviewer, antes de qualquer pedido de aceite. A ligação do ISRC ao fluxo depende das ISSUE-218/219; o critério 3 permanece aberto. Prioridade e teto de importação pertencem à ISSUE-213. Cache negativo persistente não foi implementado; proposta registrada no comentário da issue.

**Alertas:** a regra da ISSUE-215 foi escolhida na mesma amostra de 35 escutas e seus resultados são ordem de grandeza, não precisão esperada. Não houve consulta real ao MusicBrainz. `make check` passou com 13/13 percursos; race passou nos pacotes solicitados, incluindo o teste de orçamento sem filtro. Mutações que removeram o filtro de duração e aceitaram o primeiro candidato fizeram os testes falharem.

**Advisor:** examinar a regra de aceitação, ambiguidades (inclusive IDs duplicados), uso exclusivo do Budget, escape Lucene e o tratamento de `length` desconhecido: rejeitado quando a escuta tem duração, mas permitido quando ela não tem, ainda exigindo álbum e créditos. A issue permanece Em andamento.

## Depois do parecer do Advisor (27/11/2026)

Aplicadas as melhorias 1, 2 e 3 do `PARECER-ADVISOR.md` e removida a linha morta do escape; a melhoria 4 (ambiguidade por truncamento só se o último resultado tem score >= 95) ficou de fora, porque afrouxa a regra.

- **Normalização** (`normalizedMetadata`): volta à comparação medida na ISSUE-215, com casefold e espaços colapsados, sem remover pontuação. «Help!» e «Help» voltam a ser álbuns diferentes.
- **Título só de pontuação** (`hasWordCharacter`): devolve Absent sem consultar, porque o analisador descarta a frase e o título não é rechecado localmente.
- **Testes novos:** mesmo MBID com caixa diferente (com um UUID que tem letras hexadecimais), página de 99 contra 100 resultados sem `count`, `Snapshot.Recording` divergente do candidato, falha que não entra no cache e título só de pontuação, mais álbum com pontuação.
- **Mutações:** cada regra nova foi quebrada de propósito e o teste correspondente falhou (dedupe por UUID canônico, limite de 100, conflito com gravação conhecida, falha no cache, guarda de título, normalização com pontuação). A primeira versão do teste de caixa não pegava a mutação, porque o UUID de teste só tinha dígitos; foi corrigida.
- **Verificação:** `make check` com 13 de 13 percursos e `go test -race` em `internal/enrichment`, `internal/mbz` e `internal/db/sqlite`.

**Ainda em aberto e registrado:** a consulta inclui `release:"..."` e a 215 mediu sem ele, então a cobertura medida não vale diretamente; vale repetir as 35 consultas com essa cláusula (35 requisições) quando houver token.
