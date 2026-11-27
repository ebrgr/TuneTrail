# Parecer do Advisor da ISSUE-216

Modo A sobre o commit `74b60ee`, de 27/11/2026, por Opus com esforço medium, acionado pelo usuário. Executado como subagente de leitura; o coordenador registrou o parecer.

**Parecer do Advisor: APROVADO.** Sem falha crítica. As melhorias de teste são baratas e recomendadas antes do Reviewer.

## Verificado pelo Advisor
`go vet` limpo; `go test -race ./internal/enrichment/`; `TestMusicBrainz*` e `TestEnrichmentChain*` em `internal/db/sqlite`; `TestEnrichmentRegistry*` em `engine`. Também conferiu o script da 215, `queue/queue.go` e `internal/db/sqlite/enrichment.go`.

## Suspeitas do coordenador, uma a uma
- **(1a) `release:"..."` na consulta (`musicbrainz.go:223`): procede, os números da 215 não valem diretamente.** A mudança vai na direção segura: a frase de release só tira candidatos que a checagem local de álbum (`:136-145`) já rejeitaria, e elimina ausências falsas que a 215 sofria com a página de 100 resultados truncada (o script marcava `truncated` mas não usava isso na regra). Não foi medido o efeito no score, que é relativo ao primeiro resultado; deve ser pequeno, porque na 215 os limiares 90, 95 e 100 deram o mesmo desfecho. Sugestão: registrar na issue ou repetir as 35 consultas com a cláusula de release.
- **(1b) Escuta sem álbum vira Absent (`:202`, `:133`): confere.** A amostra da 215 tinha 100% de escutas com álbum, então o caso nunca foi medido. É a leitura conservadora e custa só cobertura.
- **(1c) Candidato sem `length`: não é problema.** Bate com o RELATORIO §9 e com o script.
- **(1d) Lista de releases truncada: não é problema.** Só pode gerar Absent falso, nunca um MBID errado.
- **(2) Ambiguidade: confirmada.** Truncamento (`len>=100 || Count>len`, `:235`) avaliado antes dos matches vira Ambiguous; dedupe por UUID canônico (`:231`); Ambiguous e Failed não gravam; Failed retorna antes do cache; conflito de gravação via Snapshot ou Prior vira Absent. Conflito de artista conhecido elimina só aquele candidato.
- **(3) Budget:** toda chamada passa por `budget.Do` (`:95`); redirecionamento bloqueado numa cópia do cliente; corpo limitado a 8 MiB; cancelamento funciona na fila e no HTTP; orçamento não foi ampliado.
- **(4) Escape Lucene: correto.** `\` e `"` escapados; `&&` e `||` viram `\&\&` e `\|\|`; espaço, quebra de linha e Unicode passam como texto dentro das aspas. O espaço no conjunto de `:67` é anulado por `r != ' '`, código morto. Título enorme gera 414 e Failed sem cache.
- **(5) Falha transitória: aceitável na 216.** Um Failed do MusicBrainz não interrompe a cadeia, que segue para o ListenBrainz (`enrichment.go:84-90`). O ritmo já é limitado pela fila de 1 req/s, então um disjuntor não protegeria o servidor nem recuperaria dados. O risco real é outro: com a fila cheia por uma rajada do resolvedor, a espera passa dos 20 s, cada Lookup vira Failed e o job fecha. Pertence à ISSUE-213 (prioridade e reprocessamento de jobs `failed`).
- **(6) Privacidade:** só título, artista e álbum vão na consulta; a duração nem é enviada, só comparada localmente; `User-Agent` de `cfg.UserAgent()`; a chave de cache inclui perfil e Prior.
- **(7) Mutações do Codex: quebram a regra de verdade.**
- **(8) Regras do projeto:** comentários em inglês, CHANGELOG, docs, prefixo, escopo respeitados.

## Melhorias
1. **Testes que faltam**, hoje uma mutação passa sem ser pega: (a) o mesmo MBID duas vezes, com caixa diferente, deve dar Found; (b) 100 gravações com `count` ausente devem dar Ambiguous; (c) `Snapshot.Recording` válido e divergente do candidato deve dar Absent (`:241`); (d) Failed não entra no cache: uma segunda chamada deve repetir o `budget.Do`.
2. **`normalizedMetadata` (`:76-90`) remove pontuação**, enquanto o `norm` da 215 só faz casefold e colapsa espaços: a comparação de álbum ficou mais frouxa que a regra medida («Help!» passa a igualar «Help»). Risco baixo, mas ou se registra a divergência, ou se segue a regra medida.
3. **Título só de pontuação** («?», «...») normaliza para vazio; a busca vira só artista E release, e basta uma faixa dentro de ±5 s para ser aceita. Proteção barata: Absent quando `normalizedMetadata(s.Title) == ""`, junto da guarda de `:202`.
4. **Opcional:** considerar ambíguo por truncamento só quando o último resultado da página ainda tem score >= 95, o que ganha cobertura.

## Nota sobre a ISSUE-214
O lease de 60 s é renovado a cada etapa (`enrichment.go:386`) e cobre os timeouts de 20 s e 7 s. A gravação acontece na mesma transação, depois do fence; uma escrita atrasada com o lease vencido falha o job em vez de gravar. O ListenBrainz relê a evidência persistida e confere de novo sob o lock. Custo aceito: as respostas Absent não são gravadas, então um job recuperado pergunta de novo ao MusicBrainz.

## Não verificado pelo Advisor
O comportamento real do servidor MusicBrainz (análise de frase só de pontuação, se a lista `releases` da busca vem completa, limite de tamanho de URL); uma corrida de renovação do lease sob carga real; o `make check` completo.
