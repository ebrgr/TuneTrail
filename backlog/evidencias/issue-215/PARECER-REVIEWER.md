# Parecer do Reviewer da ISSUE-215

Revisão de 26/11/2026 do diff `main...HEAD` da branch `spike/issue-215-estudo-medir-musicbrainz-por-nome` (2 commits do agente DeepSeek, sem código de produção). Executada por Claude, que despachou a tarefa, sem delegar de novo.

**Resultado: itens obrigatórios abertos. Não aceitar antes de corrigir os itens 1 a 3.**

## Itens obrigatórios

1. `[BUG]` `medir_issue_215.py:196` e `:214`, e `RELATORIO.md:38` e `:204-206`: o script lê `submission_client` em `track_metadata`, mas o campo fica em `track_metadata.additional_info`, então a contagem de escutas do Navidrome dá sempre zero. O relatório então afirma como fato que «nenhuma escuta veio do Navidrome» e que «a preferência por `submission_client = navidrome` não pôde ser aplicada», o que é falso. Conferido com uma consulta de leitura ao histórico (só contagens por nome de cliente): das 1.299 escutas varridas, **166 têm `submission_client = navidrome`** e **1.133 têm `listenbrainz`**; as **39 escutas com `recording_mbid` do cliente são todas do Navidrome**. A amostra de 35 combinações veio, portanto, do cliente que importa, e o relatório diz o contrário. Correção: ler o campo em `additional_info`, refazer a contagem e reescrever a tabela da seção 2 e o primeiro e o segundo marcadores da seção 11.
2. `[BUG]` `RELATORIO.md:225` e `medir_issue_215.py:42`: o caminho do arquivo do token contém o usuário do macOS (`/Users/<usuário>/.cache/...`). É dado pessoal num arquivo versionado. Correção: usar `~/.cache/tunetrail-215/lb_token` e expandir com `Path.expanduser()`.
3. `[RISCO]` `medir_issue_215.py:544-550` e `RELATORIO.md:178-181`: a regra «proposta» é escolhida pelo máximo de precisão **na mesma amostra** em que é avaliada (12 combinações de tolerância e limiar sobre 35 escutas), e o relatório apresenta os 100% de precisão como resultado da regra. É seleção dentro da amostra: a diferença entre ±2 s e ±5 s é uma única escuta. Correção: dizer isso na seção 9 e tratar os números como ordem de grandeza e direção, não como desempenho esperado.

## Riscos a registrar

4. `[RISCO]` `RELATORIO.md:42-46`: a amostra só inclui escutas que **já trazem** MBID do cliente (39 de 1.299), isto é, faixas bem etiquetadas, que o MusicBrainz conhece bem. A população que a ISSUE-216 quer atender são as escutas **sem** MBID: pelas contagens acima, 127 das 166 do Navidrome (76,5%), justamente as que não têm referência. Os acertos de 80% e 100% provavelmente superestimam o desempenho nessas faixas. Falta dizer isso no relatório.
5. `[RISCO]` `RELATORIO.md:150-153` e `:185-188`: «100% dos ISRCs levam à gravação de referência» é uma tautologia, porque os ISRCs saem da própria gravação de referência; e «35 de 35 ganham MBID sem ambiguidade» conta uma gravação com ao menos um ISRC não ambíguo, enquanto uma escuta traz um só ISRC. A métrica que vale é a de unicidade (47 de 49, 95,9%). A recomendação de consultar o ISRC **antes** da busca por nome se apoia nessa medição, que não inclui ISRC vindo do Navidrome.
6. `[RISCO]` `RELATORIO.md:52-53` e `:192-194`: o ListenBrainz aparece com 100% contra 80% do MusicBrainz, com n = 35 e a seleção do item 4. A ordem recomendada (MusicBrainz primeiro, ListenBrainz depois) segue a decisão de abertos antes de contas, e não os números; vale dizer isso, para não parecer que os dados a sustentam.

## Verificado e correto

- Sem token nem UUID em nenhum arquivo ou commit da branch; o relatório e o script não contêm MBIDs, nomes de artistas, faixas, álbuns nem usuário (só o caminho do item 2). Dados brutos em `/tmp/issue215-raw` foram apagados.
- O MBID de referência é só o `recording_mbid` enviado pelo cliente (`:199`); o do mapeador do ListenBrainz não entra como referência, e a referência não é usada para escolher candidatos (só para classificar o desfecho).
- 1 requisição por segundo no MusicBrainz, `User-Agent` correto, espera em 429 e 503, semente fixa para reproduzir.
- Os números das tabelas são coerentes entre si (28 acertos, 6 ambíguas e 1 ausente somam 35; o cruzamento fecha em 35 e 7).
- Sem parecer de revisão escrito pelo próprio agente; commits com o prefixo `ISSUE-215: `; só 3 arquivos no diff.

## Não verificado

Não reexecutei a medição: o MusicBrainz estava com 503 intermitentes e a execução leva cerca de 20 minutos. A única conferência independente foi a contagem por cliente acima, feita com uma consulta de leitura ao histórico.

## Reauditoria de 26/11/2026 (commits `54755bb` e `44a7180`)

Reauditado só o que mudou depois do parecer.

**Corrigidos:**
1. Leitura de `submission_client` em `additional_info`. A contagem do script bate com a do Reviewer (166 do Navidrome, 1.133 do ListenBrainz, 39 com `recording_mbid` do cliente, todas do Navidrome), e a medição foi reexecutada com a mesma semente. Os números mudaram e o relatório foi atualizado por inteiro: MusicBrainz com a regra 27 acertos (77,1%), 6 ambíguas e 2 ausentes; ListenBrainz 35 de 35. As tabelas fecham entre si (27 + 6 + 2 = 35; 27 + 8 = 35 no cruzamento; 27 + 5 = 32 de 35 com a referência entre os candidatos).
2. O caminho do token usa `~/.cache/...` com `expanduser`. Sem usuário do macOS, token, UUID nem MBID nos arquivos.
3. A seção 9 diz que a regra foi escolhida na mesma amostra em que foi avaliada, não apresenta os 100% como desempenho esperado e informa que 6 escutas separam ±2 s de ±5 s, sendo 1 a que decide.
4. A seção 11 registra o viés de seleção, com 127 das 166 escutas do Navidrome (76,5%) sem `recording_mbid`; a tautologia das linhas de 100% do ISRC; e que a ordem das fontes segue o doc-048, não os números. As frases do relatório foram revisadas para não afirmar mais do que os dados permitem.

**Restam dois pontos menores:**
- `[RISCO]` `RELATORIO.md:64-65` e `:235-236` ainda dizem que o MusicBrainz respondeu 503 e que 2 consultas foram repetidas. Parece texto da primeira rodada: a latência da rodada nova (máximo de 2,8 s, p95 de 2 s) não mostra isso, e o script não registra tentativas por consulta, então a frase não se sustenta com os dados. Correção: remover ou registrar as tentativas no script.
- `[RISCO]` `RELATORIO.md:55-58` e `:235`: a frase de que o lookup do ListenBrainz «favorece essa fonte por construção» é uma hipótese plausível, não um fato medido. Basta escrever «possivelmente».

**Não verificado:** não reexecutei a medição nem conferi o «6 escutas mudam de desfecho», porque os dados brutos foram apagados, como pedido.

**Resultado: sem itens obrigatórios abertos. Aceitável com os dois pontos acima registrados, ou com uma correção de uma linha.**

## Fechamento (26/11/2026)

Por decisão do usuário, os dois pontos menores foram corrigidos pelo Reviewer, e não pelo agente: o trecho sobre 503 do MusicBrainz foi trocado por uma nota de que o script não registra tentativas por consulta (também a afirmação de que o ListenBrainz não teve 429 ou 503, que o script não permite provar), e «por construção» virou «possivelmente». Os números não foram tocados. Sem itens abertos.
