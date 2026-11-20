# ISSUE-134 — Ensaio da E3

O ensaio usou o binário da worktree baseada na v0.45 e um binário compilado da tag v0.44. Todos os dados e servidores foram descartáveis. O perfil v9 de origem foi atualizado para v11. Outro perfil conservou **300.000 recibos de sombra** enquanto o perfil de teste resolvia as escutas e os arquivos. MusicBrainz e relay ficaram desligados.

| Medida | Resultado observado |
| --- | ---: |
| Upgrade v9→v11 até o endpoint de saúde | 9,086 s |
| Backup automático antes do upgrade | 69.079.040 bytes |
| Escutas aceitas na primeira onda, em seis origens | 101 |
| Primeira onda, do início da submissão à visibilidade de todas as escutas | 3,922 s |
| Do último ACK de upload à visibilidade das 101 escutas | 3.874,6 ms |
| Importação Last.fm sobreposta com digest novo | 20 pendências e zero escutas adicionais |
| Fixture direta de três chamadas ao resolvedor em SQLite local | 2,35 ms somados |
| Fixture direta, primeiro item após admissão | 1,03 ms até o commit visível |

A primeira onda inclui dois envios HTTP concorrentes com a mesma chave e 20 itens de cada um dos cinco formatos de arquivo. A chave concorrente produziu um recibo. Seu reenvio com metadados ampliados produziu uma observação. O reenvio do mesmo arquivo conservou os recibos. A importação sobreposta mudou os bytes do arquivo e conservou os mesmos itens.

Uma falha de COMMIT induzida por FK adiada deixou o recibo em retry e zero escutas para o item. O processo foi encerrado e, após remover a falha e liberar o agendamento, reiniciou com o mestre desligado. O worker aceitou o recibo restante sem arquivo de staging. Uma admissão nova recebeu HTTP 503. A base terminou com 102 aceitos, 20 pendências e os 300.000 recibos do perfil de sombra.

O replay com controles desligados comparou 18 escutas, 18 recibos de sombra e exportação com a v0.44. O backup v9 abriu e repetiu o mesmo resultado no binário da tag.

O `EXPLAIN QUERY PLAN` da seleção de ocorrências reportou `MULTI-INDEX OR`, com dois ramos usando `idx_occurrences_release` e um ramo usando o índice único de `track_mbid`. A gravação foi buscada pela chave primária. A comparação de créditos usou os índices de aliases e vínculos, e a ordenação usou uma B-tree temporária. A busca textual de lançamento começa pelo alias.

Comando de reprodução:

```sh
python3 scripts/rehearse_issue134_e3.py --binary /caminho/v11 --baseline /caminho/v044
```

Os tempos são observações de um ensaio local e incluem a cadência do worker na execução HTTP. A fixture direta mede trabalho local, sem espera de fila, catálogo remoto ou carga concorrente. O ensaio usou uma rajada de 101 escutas com um ledger grande e não estabelece limites de produção. A E4 mede os casos de fragmentação, o piloto e a carga sustentada após o Reviewer.
