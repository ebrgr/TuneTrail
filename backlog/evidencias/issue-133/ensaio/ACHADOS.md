# ISSUE-133 — Passo 7 interrompido em fixture sintética

Decisão de 2026-11-19. O ensaio deve usar dados sintéticos. Caso seja encontrado defeito na 008 ou no código, registrar a reprodução e parar para decisão do usuário.

## Exclusão de faixa varre duas vezes o ledger

O ensaio confirmou a ressalva R1 do Advisor para `ae09b13`. Na 008, `ingest_events.occurrence_id` e `recording_id` referenciam seus pais com ON DELETE SET NULL. Os índices do ledger não cobrem essas duas colunas. O SQLite planeja uma varredura integral do ledger para cada uma dessas FKs, mesmo quando todos os recibos têm IDs nulos, como na sombra da ISSUE-171.

O caso mínimo tem uma faixa, um lançamento, um perfil e um recibo sintético desvinculado. O binário v0.42 cria o schema v7. O reproduzidor grava o recibo em WAL com autocheckpoint desabilitado, copia DB/WAL/SHM antes da primeira abertura da cópia e abre somente a cópia com o binário da branch para aplicar a 008. Os hashes físicos dos três componentes da origem sintética foram conferidos em memória e permaneceram iguais.

| Verificação em v8 | Resultado |
| --- | --- |
| Recibos totais / vinculados | 1 / 0 |
| Plano de exclusão por PK de occurrences com FK ligada | `SCAN ingest_events` |
| Plano de exclusão por PK de recordings com FK ligada | `SCAN ingest_events` |
| Exclusão da faixa dentro de BEGIN IMMEDIATE | A ocorrência e a gravação órfã foram removidas. O recibo permaneceu. A operação foi revertida ao fim da medição. |
| Integridade e FKs da cópia migrada antes da exclusão | `integrity_check=ok`, `foreign_key_check` vazio |

O tempo pontual da exclusão mínima foi 0,106 ms, com SQLite 3.54.1 para os planos e a exclusão. Isso mede uma linha e não estima o custo em bases maiores. A evidência do defeito é o plano de execução: excluir uma faixa ou remover a origem em um merge pode fazer o trabalho crescer com todo o ledger enquanto a transação mantém o lock de escrita. Exclusões em lote repetem essas verificações por entidade removida.

## Reprodução

Compile os executáveis com `go build -o <destino> ./cmd/api` nos fontes de v0.42 e de `ae09b13`, em diretórios separados. Execute no checkout da ISSUE-133:

```sh
python3 scripts/repro_issue133_delete_scan.py \
  --v7 /caminho/bin-v0.42 --v8 /caminho/bin-ae09b13
```

O script aceita `--receipts` para variar o número de recibos sintéticos. Ele cria e remove o próprio diretório temporário fora do repositório e aceita somente caminhos de executáveis.

O passo 7 parou no primeiro defeito confirmado. Permanecem pendentes o gerador de catálogo e escutas, os ensaios de 30 mil e 300 mil escutas, o relatório de carga e o fechamento dos AC1 e AC4. A migração 008 e os arquivos em revisão não foram alterados. Nenhum banco real ou de usuário foi acessado.

## 2026-11-19 — Enum de origem de escuta requer decisão

Na rodada de correções autorizada por “aplique tudo”, a instrução para fechar o CHECK de `listens.origin` remete aos valores previstos no doc-039. O contrato identifica os eventos migrados como `legacy`, mas não enumera os valores de origem de eventos após o corte. O único outro literal de origem no doc-039, `unknown`, é descrito na seção de resolução de ocorrência para lançamento ausente. O doc-040 separa tipo de fonte e operação no envelope do recibo. Um evento pode ter mais de um recibo, então esses campos também não formam um enum inequívoco de origem do evento.

Antes de fixar o CHECK na 008, é preciso definir se `origin` distingue apenas evento legado de evento resolvido pelo pipeline, ou se distingue os caminhos pelos quais um evento foi criado. Uma opção é fixar `legacy` e `pipeline`, mantendo o tipo da fonte e a operação nos recibos. Uma lista fechada incorreta poderia impedir a escrita na ativação. Corrigi-la depois da publicação exigiria nova reconstrução de `listens`, que na v8 é pai do ledger e da outbox.

A execução foi interrompida conforme o ponto de parada solicitado. As correções na 008, o ensaio de carga e o fechamento dos AC1 e AC4 aguardam essa decisão. Os pareceres paralelos foram preservados neste commit.
