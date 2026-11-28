# ISSUE-218 — Correções pedidas após o parecer do Reviewer

27/11/2026. O parecer está em `PARECER-REVIEWER.md` (**REPROVADO**). Leia-o antes de editar.

Para cada item abaixo: **avalie primeiro** (reproduza com um teste que falhe no código atual), depois corrija. Se discordar de um item, não o altere em silêncio: registre a evidência no `HANDOFF.md`.

## Itens obrigatórios

1. **Identidade não pode depender dos detalhes** (`internal/enrichment/subsonic.go`, `lookup`).
   - Falha ou divergência em `getSong` ou `getAlbum` mantém as declarações de MBID já obtidas.
   - Divergência no álbum descarta só os valores do álbum; os da faixa continuam.
   - Sem nenhuma declaração, o resultado segue `failed` ou `ambiguous` como hoje.
   - Um resultado parcial por falha não pode ficar no cache de 10 minutos: a próxima escuta tem de tentar de novo.
2. **ISRC desconhecido** (`internal/enrichment/musicbrainz.go`). O 404 de `/ws/2/isrc/` é ausência. Ausência ou falha da consulta por ISRC segue para a busca por nome. O 404 dos outros caminhos continua sendo erro. Se a consulta por ISRC falhou e a busca por nome não achou nada, o resultado não pode ser guardado como ausência.
3. **Faixa com mais de um ISRC.** Consulte-os (com um limite pequeno e documentado: o orçamento é de 1 chamada/s e o lease é de 60 s) e una as gravações distintas: uma só é `found`; mais de uma é `ambiguous`; nenhuma segue para a busca por nome. Se uma das consultas falhar, não declare a partir de evidência parcial. A ordem das consultas deve ser determinística. O resultado por ISRC deve usar o cache, como o da busca por nome.
4. **MBIDs de artista.** Quando o ISRC encontra a gravação, os artistas que faltam continuam sendo declarados, com as mesmas verificações da busca por nome (nome do crédito igual, único, sem conflito com ID conhecido). Confirme na documentação ou na API pública qual `inc` o endpoint aceita, sem enviar dados pessoais; teste com servidor simulado.
5. **Gatilhos proporcionais à biblioteca** (`db/migrations_sqlite/026_descriptive_metadata.sql`). Reescreva `trg_descriptive_scope_insert`, `trg_descriptive_listen_delete`, `trg_descriptive_occurrence_delete` e `trg_descriptive_track_delete` com uma condição por tipo de entidade, apoiada em índice. As regras atuais continuam valendo (vínculo só com escuta que o sustente; poda quando a última some). A migração 026 não foi publicada: edite-a no lugar.
   - Meça de novo com `medir_gatilhos.py` (atualize `esquema-gatilhos.sql`) em 1 mil, 10 mil e 100 mil escutas, incluindo a exclusão de ocorrência e de faixa e a exclusão em lote de um perfil. O custo por escuta não pode crescer de forma proporcional à biblioteca. Mostre o `EXPLAIN QUERY PLAN` das consultas dos gatilhos.
   - Se criar índices, repita a medição de bytes por escuta (1 / 3 / 10). Se o caso de 1 escuta por faixa passar de **700 B**, pare e relate antes de seguir.

## Regra nova (aprovada pelo usuário)

6. **Consultar o Navidrome só quando faltar algo.** Escuta sem nenhum ID pendente cuja faixa (mesma ocorrência, mesmo perfil, mesma fonte e instância) já tem metadado descritivo não faz nenhuma chamada. Com ID pendente, ou sem metadado, o comportamento segue o dos itens acima. A decisão é do armazenamento, que conhece o banco; passe-a ao provedor sem alterar a chave de cache existente. Teste contando as chamadas ao servidor simulado. Atualizar metadados já gravados fica fora desta issue; registre essa limitação no `HANDOFF.md`. Registre também o caso de faixa para a qual o servidor não devolve metadado de faixa (será consultada de novo).

## Riscos para avaliar

Responda cada um no `HANDOFF.md`; altere só se houver motivo demonstrado por teste:

- `finishClaimedEnrichment` passou a exigir lease não vencido: o que acontece com o trabalho que vence no meio da execução, agora que cada trabalho faz mais chamadas? Há como um trabalho nunca terminar?
- `ResolveNextEnrichmentChain` testa o tipo concreto `*enrichment.Subsonic`; avalie trocar por uma propriedade do provedor.

## Verificação e limites

- Um teste por item, conferido por mutação (a mutação precisa compilar).
- `make check`, e `go test -race` com `CGO_ENABLED=1` nos pacotes alterados (`TUNETRAIL_WEB_DIR=$PWD/assets TUNETRAIL_LANGS_DIR=$PWD/langs`).
- Atualize `HANDOFF.md`, `MEDICAO.md` e, se o comportamento visível mudar, o `CHANGELOG.md` (em inglês). Comentários de código em inglês.
- Commits com o prefixo `ISSUE-218: `. Sem push, merge, publish ou `Done`. Não invoque outro agente nem subagente.
- Não grave credenciais, tokens ou dados pessoais em arquivos, logs ou commits.
