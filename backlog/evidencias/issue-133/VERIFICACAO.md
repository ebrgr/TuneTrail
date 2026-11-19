# ISSUE-133 — Verificação sintética da migração 008

Passos 1 a 6 do plano. Rebase sobre a main na base v0.42 (`6007a8d`), sem conflito. A 008 segue a 007 de account_tokens. Comparação com a base confirmou as migrations 001–007 intactas. Commits de implementação `5c863ee` e de testes `52248c1`.

Todos os bancos usados nesta etapa foram fixtures sintéticas ou cópias delas. O ensaio com snapshot autorizado e a revisão do passo 8 estão pendentes. A mudança é interna e mantém o comportamento das rotas e dos escritores legados. O changelog segue o critério de mudanças visíveis ao usuário.

## Evidências por critério

| AC | Evidência sintética | O que falta |
| --- | --- | --- |
| 1 | `TestRecordingIdentityUpgrade` migra v7 → v8 com dois perfis, três escutas (2 e 1), aliases primários e secundários, MBIDs nulos, FKs e atribuições legadas. Compara digests de todas as tabelas anteriores nas respectivas colunas de origem. A 008 testa contagens por perfil, EXCEPT nos dois sentidos e FK check na própria transação. | Repetir em cópia do snapshot autorizado, com contagens e resultados agregados. |
| 2 | Duas faixas chamadas Intro geram gravações distintas. IDs 1 e 70 mapeiam para as ocorrências correspondentes. A ponte também foi exercitada com INSERT, atualização de release/MBID, exclusão e limpeza de gravação órfã. | — |
| 3 | Comparação HTTP entre os binários v0.42 e da branch, sobre o mesmo histórico sintético antes/depois da 008, retornou respostas idênticas para `/apis/web/v1/profiles/{slug}/track/1` e sua rota de aliases. Os testes de ponte e as jornadas de leitura, exportação e permissões exercitam os caminhos legados. | — |
| 4 | FS de teste do Goose força falha de CHECK no fim da 008. Versão 7, layout e digests permanecem íntegros. A nova tentativa aplica a 008 e uma segunda execução mantém os IDs. A restauração independente comprova que VACUUM INTO inclui escuta confirmada no WAL. Binários compilados dos tags v0.41 e v0.42 recusaram o banco sintético v8, preservando versão e digests legados. | Ensaio de backup e restauração com a cópia autorizada. |
| 5 | Eventos legados recebem ID opaco, com FK composta do ledger e outbox. Recibos de sombra, variantes e detalhes preservam seus dados e a decisão original. A fixture recusa vínculos entre perfis e verifica accepted/duplicate com evento; as guardas de revisão e outbox bloqueiam INSERT no schema entregue. | — |
| 6 | Fixture pós-corte remove um evento por perfil/ID e exercita rollback e cascatas sobre recibo, revisão, detalhes, observação e outbox. Outro evento no mesmo segundo, outro perfil e a configuração da fonte permanecem. O mesmo item pode reutilizar as três chaves de recibo e recebe um event_id novo, pelas primitivas SQLite. | A reimportação pelo pipeline ativo pertence à 081/134. |
| 7 | O UPSERT legado opera com o índice único completo e mantém a regra de colisão do merge. Guardas recusam ocorrências independentes, mudança de tracks.id e escrita com origem nova. Uma fixture isolada retira as guardas para demonstrar DROP INDEX, dois eventos no mesmo segundo, múltiplas ocorrências e lançamento nulo. O cursor por (instante, event_id) continua após excluir a âncora e respeita o perfil. | Integração dos leitores e escritores na ativação coordenada com a 135. |

A exclusão de lançamento foi exercitada durante a ponte e após a retirada das guardas em fixture. Durante a ponte, a cascata do catálogo mantém o comportamento legado. A ocorrência independente no cenário pós-corte sobrevive à exclusão do lançamento com release_id nulo. A validação reversa também recusa índice inesperado no escopo da 008 antes de alterar a origem v7 e ao abrir uma v8.

## Execução local

- `make check` com assets da worktree, incluindo 13 jornadas E2E.
- `CGO_ENABLED=1 go test -race -count=1 ./internal/db/sqlite`.
- `python3 scripts/validate_ingest_shadow.py` sem opção de origem. A regra shadow-v2 manteve as decisões esperadas no replay sintético da 171, com quatro itens encaminhados a Revisar.
- Comparação HTTP v7/v8 e recusa do v8 por binários anteriores em diretório temporário, usando a fixture de `scripts/validate_multiuser.py`.
- Graphify AST atualizado e `git diff --check` sem erros.

## Ponto de parada

A ISSUE-133 segue Em andamento. AC1 e AC4 ficam abertos até o ensaio. O passo 7 exige nova instrução e indicação explícita do snapshot pelo usuário, com serviço parado e espaço livre de pelo menos três vezes o tamanho do banco. A revisão do passo 8 também fica pendente.
