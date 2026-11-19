# ISSUE-133 — Ensaio sintético da 008 corrigida

Foram usados apenas bancos sintéticos gerados fora do repositório. As fixtures foram criadas pelo bootstrap do binário v0.42 no schema v7, com gerador de semente 133 e serviço parado ao copiar DB/WAL/SHM. A origem não foi aberta pelo binário v8. Os hashes físicos dos três componentes da origem foram comparados em memória antes e depois. Um snapshot real permanece uma verificação adicional opcional.

## Composição

Ambas as cargas têm 4 perfis, incluindo um protegido, um vazio e um com chave/relay. O catálogo após os comandos v7 de merge e exclusão de lançamento tem 600 artistas, 1.502 lançamentos, 9.012 faixas, 11.265 créditos de faixa, 1.200 aliases de artista, 3.004 aliases de lançamento e 18.024 aliases de faixa. Há capas e artista com imagem, colaborações, títulos homônimos, uma família de edições deluxe/remaster, MBIDs presentes e nulos, uma conta admin e três pendentes. As sete integrações incluem fontes de sombra e uma configuração independente.

A v7 exige release_id e MBID de gravação não nulo único em tracks. As faixas sem lançamento verdadeiro usam um álbum substituto e os equivalentes em edições distintas usam MBID nulo. A remoção de lançamento é exercitada no v7, inclusive a cascata das suas faixas. A estrutura de ocorrência independente com lançamento nulo é testada separadamente pela fixture pós-corte `TestRecordingIdentityExactDeletion`. A migração mantém gravações separadas para faixas equivalentes sem MBID confirmado.

| Carga | Público | Protegido | Relay | Vazio | Recibos de sombra |
| --- | ---: | ---: | ---: | ---: | ---: |
| 30.000 | 10.001 | 10.000 | 9.999 | 0 | 30.000 |
| 300.000 | 100.001 | 100.000 | 99.999 | 0 | 300.000 |

Histórico distribuído de 2018 a 2023, com segundos compartilhados entre perfis, repetições em segundos distintos e clientes API/arquivo. Os recibos sintéticos contêm offsets de quatro fusos, horários estimados e decisões same/different/review, todos sem event_id. O ensaio também envia um scrobble ListenBrainz real ao servidor local sintético depois da migração.

## Resultados

- As contagens por perfil e digests das linhas de todas as tabelas legadas foram iguais após o upgrade, usando a projeção de colunas v7. Isso inclui listens, artist_tracks, track_aliases, releases e todos os campos de ingest_events. O mapa manteve o mesmo ID para cada faixa, uma gravação por ocorrência legada e IDs próprios em todas as escutas, com integridade OK e FKs vazias.
- As 34 respostas de leitura foram equivalentes via HTTP nos binários v7 e v8. A comparação cobre estatísticas, duas páginas de histórico, top de artistas/álbuns/faixas, busca, exportação, faixa e aliases. Foram ignorados apenas os campos que dependem do relógio da requisição, range.to e exported_at.
- O FS de teste do Goose injetou uma falha no último passo da 008. O banco permaneceu v7, os digests não mudaram e a nova tentativa pelo bootstrap real funcionou. A injeção levou 0,78 s na carga menor e 9,31 s na maior.
- O bootstrap v8 criou o backup VACUUM INTO com o histórico confirmado em WAL. O backup restaurado iniciou com v0.42, passou na verificação v7 e manteve os digests. O segundo boot em v8 manteve os dados e os event_ids. v0.41 e v0.42 recusaram v8, preservando os digests.
- Em cópia separada, POST legado fez INSERT e o reenvio deixou uma escuta. Depois foram exercitados exclusão de escuta, novo recibo de sombra pelo ListenBrainz, merge de faixa, exclusão de lançamento, exclusão do perfil protegido e PurgeAllData. A suite SQLite também testa ApplyProfileDelete na fixture `TestShadowDeletePaths`. As guardas e a tentativa de colisão explícita de ID foram testadas separadamente.
- A guarda de origem é removível. A fixture confirmou DEFAULT legacy, FK recusando origem ausente e INSERT no catálogo habilitando a nova origem após retirar a guarda, sem mudar o DDL de listens.
- Os planos das ações internas de exclusão passaram de SCAN para SEARCH por índice parcial. O índice histórico de perfil e instante foi preservado porque o EXPLAIN da consulta representativa o escolheu em v7 e em v8. [Planos antes e depois](ACHADOS.md)

## Tempo, disco e memória

Medições aproximadas em arm64. Pico de RSS do processo do binário e pico por amostragem de 15 ms dos arquivos visíveis de DB, WAL, SHM, backup e diretório temporário. Arquivos temporários que o SQLite remove ainda abertos não aparecem nessa soma. Os tempos até prontidão incluem o bootstrap e a sondagem HTTP local. MB abaixo usa base decimal.

| Medida | 30 mil | 300 mil |
| --- | ---: | ---: |
| Bootstrap v8 | 2,46 s | 24,34 s |
| RSS máximo observado | 42.768 KiB | 45.008 KiB |
| Pico visível de disco | 61,84 MB | 539,29 MB |
| Banco v7 consolidado | 17,43 MB | 144,83 MB |
| Banco v8 consolidado | 24,17 MB | 201,72 MB |
| Backup v7 VACUUM INTO | 16,58 MB | 137,65 MB |
| Páginas v7 → v8 | 4.255 → 5.902 | 35.359 → 49.248 |
| Páginas livres v7 → v8 | 0 → 0 | 0 → 0 |
| POST legado com nova escuta | 5,7 ms | 25,0 ms |
| Exclusão de escuta | 0,8 ms | 1,7 ms |
| Merge de faixa | 1,2 ms | 1,3 ms |
| Exclusão de lançamento | 0,9 ms | 1,0 ms |
| Exclusão de perfil | 0,16 s | 2,75 s |
| Limpeza global | 0,38 s | 4,95 s |
| Novo recibo pela API ListenBrainz | 15,9 ms | 77,1 ms |
| 1.000 UPSERTs legados v7 → v8 em transação revertida | 2,78 → 42,49 ms | 2,87 → 42,84 ms |

Numa rodada de 30 mil com o contador separado do WAL habilitado, o pico visível do WAL durante o bootstrap foi 20,84 MB. Na de 300 mil, o WAL v7 copiado tinha 145,63 MB e uma amostra após o boot v8 tinha 5,52 MB. O pico separado do WAL v8 não foi capturado nessa carga. O pico agregado inclui o WAL em ambas.

O microbenchmark em memória para isolar apenas a FK de origem teve medianas de 8,20 ms sem a FK e 9,49 ms com ela, a cada 30 mil INSERTs. A comparação v7/v8 do UPSERT inclui geração dos IDs, novos índices e FKs, além de diferenças de cache e disco. A diferença da comparação completa não pode ser atribuída à referência de origem.

A alternativa experimental compara listens com listens_new antes do DROP e evita a cópia TEMP integral. Os mesmos digests foram preservados. A variante levou 2,89 s e 61,90 MB visíveis na carga menor, e 27,89 s e 539,36 MB na carga maior, contra 2,46 s/61,84 MB e 24,34 s/539,29 MB na implementação entregue. Sem ganho claro nesta medição, a 008 mantém o staging.

## Reprodução

Compile os binários v0.41 e v0.42 em diretórios separados a partir dos respectivos tags. Na branch da issue, execute:

```sh
go build -o /tmp/tt133-v8 ./cmd/api
go build -o /tmp/tt133-fault ./scripts/issue133_fault
python3 scripts/build_issue133_no_temp.py /tmp/tt133-no-temp
for n in 30000 300000; do
  python3 scripts/rehearse_issue133.py --v6 /tmp/tt133-v041 --v7 /tmp/tt133-v042 \
    --v8 /tmp/tt133-v8 --alternative /tmp/tt133-no-temp --fault-probe /tmp/tt133-fault \
    --listens "$n" > "/tmp/tt133-${n}.json"
done
```

A saída JSON inclui os inventários e hashes, planos, tempos e picos. Os hashes lógicos do catálogo e histórico são da projeção de dados, independentes de event_ids aleatórios. Hashes físicos do WAL e SHM variam entre execuções e são comparados dentro da mesma execução. [Saídas agregadas com hashes e planos](MEDICOES.json).
