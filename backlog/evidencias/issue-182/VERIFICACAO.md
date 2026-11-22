# Verificação da ISSUE-182

O teste usa SQLite em arquivo temporário com WAL, em um Mac com Apple M4
(darwin/arm64, Go 1.26.3). Cada execução cria 10 mil recibos estacionados em
um único perfil. O relógio mede apenas a chamada ao repositório, incluindo
a abertura e o commit da transação. A preparação, a repetição idempotente
e a verificação dos registros ficam fora do tempo medido.

```sh
CGO_ENABLED=0 go test ./internal/db/sqlite -run '^TestIngestReprocessAllLargeQueue$' -v -count=5
```

| Execução | Tempo |
| --- | ---: |
| 1 | 115,09 ms |
| 2 | 115,59 ms |
| 3 | 116,25 ms |
| 4 | 116,69 ms |
| 5 | 116,20 ms |

Mediana de 116,20 ms. Cada instrução trata até 500 recibos. A transação
mantém o lock de escrita até o commit para impedir que o resolvedor intercale
claims entre lotes. O tempo pode crescer com filas maiores ou em armazenamento
mais lento. A medição não inclui latência HTTP ou contenção.

## Verificações

- `make check` com `TUNETRAIL_WEB_DIR=$PWD/assets` passou, incluindo 13 cenários E2E.
- Testes direcionados com `-race` passaram em `internal/db/sqlite` e `engine`.
- O relógio de teste expira cada lease durante a resolução. O recibo estaciona
  ao ficar elegível depois do 25º claim, sem emitir outro claim. Um novo ciclo
  também permite 25 claims e mantém a contagem vitalícia.
- Os testes verificam isolamento entre perfis, POST e CSRF, repetição,
  concorrência com o resolvedor, auditoria unitária e em massa, backup v17 e
  guarda de downgrade.
- A referência ISSUE-075 regenerada coincide nos 176 arquivos. A tradução
  atualiza o hash de geração nos HTMLs públicos. `health.css` tem 68 linhas
  e usa apenas os breakpoints 40, 64 e 90 rem.

## Correções após o parecer

A mesma medição, com uma revisão de auditoria por recibo, levou 152,05 ms,
158,17 ms, 150,22 ms, 145,35 ms e 146,42 ms. A mediana foi de 150,22 ms para
10 mil recibos. Os lotes de até 500 continuam na mesma transação, com lock
de escrita até o commit.

- A tentativa usa um prazo de 100 s dentro da lease de 120 s. A prova de
  ocorrência mantém seu prazo de 20 s. O teste simula um MusicBrainz que
  bloqueia até o contexto expirar e cobre as quatro esperas, o estacionamento
  na quinta falha e o processamento do recibo seguinte. O teste de shutdown
  verifica a recuperação da lease sem consumir uma tentativa do orçamento.
- O teste de leases expiradas cobre 25 claims no ciclo inicial e cinco após
  reprocessar um `lease_exhausted`. O teste de auditoria cobre também o
  orçamento cheio de `retry_exhausted`, o isolamento entre perfis e a
  ausência de duplicação ao repetir a ação.
- A auditoria usa `author='user'` e razão `reprocessed` com a decisão
  não final `review`, permitida pelo CHECK existente. Cada ação incrementa
  `resolution_version` e grava `evidence.decided_by` e `created_at` na
  mesma transação que atualiza o último reprocessamento do recibo.
- Após as correções, `make check` passou com 13 cenários E2E. Os testes
  direcionados com `-race` passaram em `internal/db/sqlite` e `engine`.
  A comparação da referência ISSUE-075 confirmou zero diferenças.
