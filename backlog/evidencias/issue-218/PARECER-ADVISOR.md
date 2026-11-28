# ISSUE-218 — Parecer do Advisor

27/11/2026. Advisor em Opus 5.5 (ferramenta `advisor`, pedida pelo usuário), sobre a branch até o commit `70606a3`. Conferências somente de leitura e reprodução em cópia temporária feitas pelo coordenador.

**Um item bloqueia o aceite**, salvo se o usuário o aceitar como regra.

## Bloqueio

1. **[BUG]** `internal/enrichment/musicbrainz.go`, `Lookup`, ramo do ISRC — a gravação é declarada mesmo quando um ID de artista já conhecido conflita; só as declarações de artista são retidas. A busca por nome rejeita a mesma candidata (`matches`). Reproduzido com a mesma resposta simulada: por ISRC, `found` com o MBID da gravação; por nome, `absent`. Um ISRC mal atribuído grava um MBID que o restante da cadeia passa a usar como evidência. O teste `TestReviewerISRCCreditsRejectKnownArtistConflict` fixa o comportamento atual. Correção proposta: conferir a coerência antes de declarar qualquer coisa; em conflito, descartar o resultado por ISRC e seguir para a busca por nome.

Junto, no mesmo ajuste: a guarda de prévia da versão 26 também exigir `metadata_retries`.

## Para decisão do usuário (não bloqueiam)

2. **Critério 1.** O texto pede gravar valores «só depois de associação segura da faixa e do álbum». Quando só o álbum diverge, a implementação mantém os valores da faixa (gênero, ano, ISRC, BPM), como pediram as instruções da primeira rodada de correções. Confirmar ou rejeitar, e anotar na tarefa.
3. **Carga residual no Navidrome.** A regra de consultar só quando faltar algo depende de metadado gravado. Escuta com todos os IDs de uma faixa ausente do Navidrome ainda chama `search3` a cada escuta, passado o cache de 10 minutos; na v0.81 não havia chamada. Proposta: tarefa própria (marca negativa durável), fora desta.
4. **`INDEXED BY` nos gatilhos da 026** cria dependência dos nomes dos índices, inclusive `sqlite_autoindex_occurrence_credits_1`: sem o índice, o comando que dispara o gatilho falha. As ISSUE-195 e ISSUE-210 vão recriar essas tabelas; os testes acusariam. Anotar nas duas.

## Menores

- `configureEnrichment` apaga as execuções concluídas trabalho a trabalho dentro de uma transação; um `DELETE` por conjunto seria mais simples.
- Com as consultas repetidas corrigidas, o índice de `metadata_retries` por `occurrence_id` não é necessário.

## Confirmado

- Lease: um executor vencido não encerra o trabalho; a recuperação é limitada a duas tentativas.
- Gatilhos: podam as mesmas linhas que os anteriores (100 mil operações, chaves estrangeiras ligadas, como em todas as conexões da aplicação) e escalam (0,06 ms por gravação e 0,05 ms por exclusão com 100 mil escutas).
- Cache do Navidrome e divergência definitiva: verificados antes e depois.
- ISRC: 404, vários códigos e créditos de artista corretos, exceto o item 1.

## Pendente

`make check` não rodou sobre o código final; a suíte completa de `internal/db/sqlite` com detector de corridas (cerca de 23 minutos) nunca rodou nesta branch.
