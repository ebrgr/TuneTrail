# Verificação da ISSUE-185

## Descarte

- A central de saúde oferece confirmação individual e em massa ao dono do
  perfil e ao root. As seções de falhas e revisão levam à confirmação do
  recibo. A lista de descarte também cobre os demais pendentes e inválidos.
- Cada POST em massa confirma os 25 recibos elegíveis mais antigos, ou os
  restantes se forem menos. O digest inclui perfil, seleção, estado, dados de
  estacionamento e versões de resolução e trabalho. O escritor revalida a
  seleção e aplica o CAS em uma única transação. Repetir um lote retorna
  conflito sem avançar a fila. Repetir um descarte individual mantém o
  registro original.
- A migração v20 cria `ingest_discards` e um índice parcial para a seleção.
  O rastro sobrevive à exclusão do recibo, pois as revisões têm FK com CASCADE.
  O rastro mantém somente o identificador, conta, data, estado anterior e
  motivos. Ele acompanha o ciclo de vida do perfil e também sai na limpeza
  global de dados. A guarda impede o downgrade.
- Intenções de relay se vinculam ao recibo pelo par perfil e ID de evento.
  A presença do evento exclui o recibo do descarte, inclusive quando há
  entrega aberta. A tela e o guia orientam a excluir a escuta e avisam que
  entregas já enviadas ao relay permanecem no destino.

## Custo de 10 mil recibos

Teste local em arquivo SQLite com WAL, macOS 15.7.3, Apple M4 e Go 1.26.3.
A fixture tem 10 mil recibos sintéticos distribuídos em 400 transações de 25
recibos para um perfil. Os tempos somam o trabalho de cada chamada de escrita,
incluindo a revalidação do digest e o commit. Preparação, geração das
pré-visualizações, latência HTTP e contenção ficam fora da medição.

| Execução | Tempo total no escritor |
| --- | ---: |
| 1 | 2,498 s |
| 2 | 2,495 s |
| 3 | 2,565 s |
| 4 | 2,492 s |
| 5 | 2,536 s |

Mediana de 2,498 s, cerca de 6,25 ms por lote em média na execução mediana.
Na interface, cada lote exige nova confirmação. O índice parcial acrescenta
custo de armazenamento e manutenção para recibos elegíveis. O tempo de escrita
depende do armazenamento e de quantos detalhes, observações e revisões
acompanham o recibo. A fixture de escala usa recibos sem esses registros
auxiliares. O teste por estado verifica a remoção dos detalhes em cascata.

```sh
CGO_ENABLED=0 go test ./internal/db/sqlite -run '^TestIngestDiscardScale$' -count=5 -v
```

## Interface

A referência ISSUE-075 foi regenerada com `make ui.reference`. O Orca abriu as
confirmações individual e em massa e confirmou a presença do aviso do relay.
O CSS tem 69 linhas e usa os breakpoints existentes de 40 e 64 rem.

| Viewport | Colunas | Largura rolável da página | Largura interna/rolável do cartão |
| --- | --- | ---: | ---: |
| 390 px | 1 × 358 px | 390 px | 356 / 356 px |
| 1100 px | 2 × 469 px | 1100 px | 467 / 467 px |
| 1500 px | 2 × 538 px | 1500 px | 536 / 536 px |

## Testes automatizados

Os testes cobrem cada estado elegível, recibos aceitos e vinculados,
intenção aberta no relay, lease ativa e expirada, isolamento entre perfis,
POST com CSRF, repetição, corrida com claim, revisão e reprocessamento,
rollback do lote, migração e guarda de downgrade. A remoção atualiza
`ingest_source_summary` e `ingest_inbox_daily`, inclusive ao remover o
último recibo de uma fonte ou restaurar seu predecessor no resumo.

```sh
TUNETRAIL_WEB_DIR=$PWD/assets go test -race ./internal/db/sqlite ./engine \
  -run 'Test(IngestDiscard(States|ProtectsListens|WriterRace|Batch|Migration)|OccurrenceDiscardPending|IngestHealth(Discard|Isolation))$' -count=1
```

O detector de corridas passou nos dois pacotes. A comparação da referência
verificou 176 arquivos sem diferenças. O grafo AST foi atualizado para a
worktree.

`TUNETRAIL_WEB_DIR=$PWD/assets make check` passou, incluindo os 13 cenários E2E.
