# ISSUE-092 — Histórico grande

## Método

Mac16,1, Apple Silicon arm64, 10 CPUs lógicas, 16 GiB, macOS 15.7.3 (24G419). Go 1.26.3 com `modernc.org/sqlite v1.49.1` no servidor v0.47. Python 3.15.2 com SQLite da biblioteca padrão para as medições SQL e os planos, identificado nos JSON de cada rodada. A ordem de escolha de índices no SQLite do Python pode divergir do driver Go. A coluna HTTP mede o servidor real e a desserialização do corpo no cliente de loopback.

O v0.42 inicializa um banco descartável por volume. O ensaio reutiliza diretamente o gerador da ISSUE-133, com semente 133, 600 artistas, 1.503 lançamentos, 9.018 faixas, colaborações, quatro perfis e recibos em sombra. Traz as escutas dos últimos 14 dias da fixture para os 14 dias anteriores a ontem para exercitar as janelas recentes da Home. O restante atravessa 2018–2023. O v0.47 migra para schema 11 com os vínculos da ponte; um perfil sem histórico serve de controle. A fixture de E4 da ISSUE-134 concentra 300 mil no mesmo perfil e exercita recibos aceitos; a carga de leitura deste spike usa o catálogo maior e a distribuição por perfil da ISSUE-133.

Amostras sequenciais de leitura (100 por operação em 300 mil e 40 em 1 milhão), três aquecimentos por operação e percentis por ordem estatística (posição `ceil(p × N)`). Com 40 amostras o p99 é o máximo observado. SQL inclui consumir todas as linhas; HTTP inclui processamento, transporte local e leitura integral. Rankings, histórico, atividade anual, estatísticas anuais e Rewind usam 2023 UTC. A página 400 pula 9.975 linhas com limite 25. A Home é medida como página completa, inclusive KPIs, painel histórico, atividade, rankings e últimas escutas. As consultas isoladas de soma de duração, contagem, artistas distintos e sequência representam os KPIs no recorte anual. A rota de atividade recente tem 252 dias. Os planos de novidade e de contagem global por perfil cobrem dois casos de agregação que atravessam o histórico.

A segunda fase adiciona somente ao banco descartável `ON listens(profile_id,listened_at,track_id)` e repete as leituras. Cada fase envia 3 aquecimentos e 100 POSTs seriais ListenBrainz com instantes inéditos para um perfil sintético no modo legado. O script verifica que todas as escritas aparecem em `listens`. A fase com índice começa com 103 escutas adicionais fora de 2023, do heatmap anual e das duas janelas de KPIs semanais. Consultas de histórico completo carregam essas 103 linhas a mais. Para recursos, o pico de RSS usa `ps` com 25 ms de pausa entre amostras; o disco do diretório inclui WAL, SHM e backups de migração. O tamanho consolidado do banco é `page_count × page_size` após checkpoint.

Reprodução a partir da raiz (o script cria e remove o diretório de dados):

```sh
work=$(mktemp -d)
mkdir "$work/v7"
git archive v0.42 | tar -x -C "$work/v7"
(cd "$work/v7" && CGO_ENABLED=0 go build -o "$work/v7-bin" ./cmd/api/main.go)
CGO_ENABLED=0 go build -o "$work/v11-bin" ./cmd/api/main.go
for n in 300000 1000000; do
  python3 scripts/benchmark_issue092.py --baseline "$work/v7-bin" \
    --binary "$work/v11-bin" --listens "$n" --output "$work/$n.json"
done
```

## Resultados

Contagens iniciais com 300 mil escutas por perfil: 100.001, 100.000, 99.999 e zero. Os pares base/com índice usam a mesma máquina e volume.

### HTTP, p50 / p95 / p99 em ms

| Carga | Consulta | Base | Com índice |
| ---: | --- | ---: | ---: |
| 300.000 | `stats_year` | 89.22 / 90.05 / 90.68 | 75.60 / 76.42 / 77.11 |
| 300.000 | `stats_all` | 296.26 / 303.04 / 314.53 | 234.29 / 257.04 / 287.15 |
| 300.000 | `top_tracks` | 15.46 / 15.94 / 16.44 | 12.82 / 15.33 / 16.84 |
| 300.000 | `top_albums` | 15.42 / 16.47 / 16.78 | 10.60 / 11.35 / 11.74 |
| 300.000 | `top_artists` | 21.16 / 31.13 / 36.05 | 15.11 / 16.65 / 22.55 |
| 300.000 | `history_first` | 2.29 / 3.04 / 3.50 | 2.20 / 2.81 / 3.27 |
| 300.000 | `history_page_400` | 18.83 / 19.54 / 19.95 | 19.22 / 19.60 / 19.89 |
| 300.000 | `heatmap_data_recent` | 25.12 / 25.57 / 25.90 | 25.49 / 26.18 / 27.48 |
| 300.000 | `heatmap_data_year` | 31.51 / 34.02 / 41.22 | 31.08 / 33.98 / 35.67 |
| 300.000 | `rewind_api` | 394.07 / 406.40 / 420.77 | 373.27 / 387.01 / 409.53 |
| 300.000 | `rewind_html` | 394.37 / 413.72 / 422.37 | 376.35 / 413.43 / 440.94 |
| 300.000 | `home_all` | 762.74 / 819.44 / 834.60 | 604.97 / 624.67 / 654.19 |
| 300.000 | `home_week` | 345.60 / 353.27 / 411.96 | 274.92 / 279.40 / 295.47 |

### SQLite isolado, p50 / p95 / p99 em ms

| Carga | Consulta | Base | Com índice |
| ---: | --- | ---: | ---: |
| 300.000 | `plays` | 0.21 / 0.22 / 0.22 | 0.21 / 0.21 / 0.21 |
| 300.000 | `minutes` | 4.31 / 4.48 / 4.51 | 2.22 / 2.36 / 2.42 |
| 300.000 | `unique_artists` | 8.94 / 9.23 / 9.29 | 7.12 / 7.26 / 7.30 |
| 300.000 | `active_days` | 15.93 / 16.57 / 16.90 | 16.48 / 16.78 / 17.57 |
| 300.000 | `longest_streak` | 2.78 / 2.82 / 2.84 | 2.81 / 2.87 / 2.89 |
| 300.000 | `activity` | 3.69 / 3.74 / 3.76 | 3.68 / 3.72 / 3.79 |
| 300.000 | `top_tracks` | 5.81 / 5.95 / 6.15 | 5.36 / 5.46 / 5.53 |
| 300.000 | `top_albums` | 6.65 / 6.77 / 6.95 | 4.31 / 4.39 / 4.42 |
| 300.000 | `top_artists` | 9.74 / 10.02 / 10.13 | 7.46 / 7.56 / 7.59 |
| 300.000 | `history_page_400` | 7.89 / 8.28 / 8.34 | 5.32 / 5.79 / 6.82 |
| 300.000 | `new_tracks` | 3.42 / 3.49 / 3.53 | 3.46 / 3.64 / 3.72 |
| 300.000 | `profile_counts` | 5.46 / 5.55 / 5.64 | 5.50 / 5.61 / 5.64 |
| 300.000 | `release_legacy` | 6.17 / 6.28 / 6.32 | 4.07 / 4.19 / 4.22 |
| 300.000 | `release_occurrence` | 6.26 / 6.33 / 6.35 | 3.87 / 3.99 / 4.02 |
| 300.000 | `recording_occurrence` | 7.88 / 8.09 / 8.17 | 5.72 / 5.80 / 5.84 |

### Ingestão e recursos

| Carga | Fase | POST p50 / p95 / p99 (ms) | DB (MB) | Pico RSS leitura / escrita (MiB) | Pico diretório leitura / escrita (MB) | Pico WAL leitura / escrita (MB) |
| ---: | --- | ---: | ---: | ---: | ---: | ---: |
| 300.000 | baseline | 66.00 / 72.25 / 105.80 | 201.83 | 77.88 / 76.81 | 339.51 / 343.76 | 0.00 / 4.13 |
| 300.000 | indexed | 65.26 / 66.15 / 68.23 | 207.25 | 74.97 / 73.56 | 344.94 / 349.25 | 0.00 / 4.19 |

## Planos de consulta

- **Intervalos e dias ativos.** As contagens por período usam busca de faixa por perfil e instante, coberta por índice. Dias ativos usam busca por perfil com teto de tempo. Nesse último caso, o teto abrange todo o histórico do perfil e produz uma linha por segundo distinto para o agrupamento em Go. A atividade e o streak também agregam inicialmente por segundo; o índice experimental preserva os planos desses caminhos. O custo cresce com a quantidade de segundos ativos mesmo sem varredura da tabela inteira.
- **Rankings.** Artistas e álbuns buscam escutas por perfil e instante e acessam créditos ou `tracks` por índice/chave primária. `GROUP BY` e `ORDER BY` usam B-trees temporárias, inclusive para `RANK()`, limite/paginação e ordenação final da projeção. Em 300 mil, o top de faixas base usou `idx_listens_legacy_identity(profile_id=?)` e percorreu o índice de todo o perfil. O índice experimental restringiu por perfil e intervalo e introduziu B-tree para agrupar por faixa. O CTE ranqueado ainda é percorrido depois.
- **Novidades do Rewind e contagem global.** `new_tracks` busca por perfil sem restringir instante, pois a primeira escuta é calculada com MIN antes de filtrar o resultado. `profile_counts` é uma leitura diagnóstica de todos os perfis e faz `SCAN` integral de um índice de cobertura. Nenhum plano implica ler todos os perfis para um ranking individual.
- **Histórico paginado.** Base: busca por faixa no índice perfil/instante, junção de título/alias e B-tree para o último termo do `ORDER BY`. O índice experimental cobre a ordem atual por instante/faixa e elimina essa B-tree, mas o `OFFSET` e a contagem total permanecem. A consulta isolada da página 400 melhora, enquanto o HTTP inclui a contagem e o enriquecimento dos itens. A ISSUE-135 muda a chave de navegação para instante/evento. O índice de cursor `idx_listens_event_cursor` já existe no schema.
- **Ponte de identidade.** Álbuns, duração, histórico e alguns filtros usam `tracks` ou suas views. O comparativo isolado troca a junção de lançamentos pela PK de `occurrences` e confere igualdade da lista agrupada. O índice experimental cobre o lado de `listens` em ambas as consultas. A troca da PK de catálogo mantém as B-trees de agrupamento e ordenação. `recording_occurrence` agrupa pela gravação com cardinalidade diferente; a fixture da 133 cria uma gravação por faixa, portanto esse tempo não representa um catálogo reconciliado com várias edições por gravação.

Os planos completos, incluindo os aliases, buscas de PK e ordenações temporárias, estão nos JSON ao lado deste relatório.

## Denominadores confirmados no binário

Em janeiro de 2023 UTC, o perfil sintético ouviu 20 vezes por dia nos dias 1, 2 e 3. O histórico acrescentou uma escuta em cada uma de 117 datas posteriores a janeiro. A faixa tinha 180 s. `/stats?year=2023&month=1` devolveu 60 escutas, 120 dias ativos e 0,50 por dia. `/summary?year=2023&month=1` devolveu 60 escutas, 1,9354838 por dia e 5 minutos por dia. Os dois endpoints receberam o mesmo intervalo e perfil.

## Ocorrência autônoma na v0.47

Em banco descartável inicializado pelo binário v0.47, sem outras escutas, executar via SQLite com `foreign_keys=ON`:

```sql
INSERT INTO recordings(id) VALUES(10);
INSERT INTO occurrences(id,recording_id,presented_title) VALUES(20,10,'Standalone fixture');
INSERT INTO occurrence_credits(occurrence_id,position,name) VALUES(20,0,'Standalone artist');
INSERT INTO listens(profile_id,track_id,listened_at,event_id)
VALUES(1,20,1672531200,'probe');
```

No prefixo `/apis/web/v1/profiles/default`, consultar com `?year=2023&tz=UTC`:

| Endpoint | Resultado observado |
| --- | --- |
| `/stats` | Uma escuta, uma faixa, um dia ativo, sequência 1, zero álbuns, zero artistas e zero minutos. |
| `/listens` | Total 1, itens vazios, próxima página indicada. |
| `/top/tracks` | Total 0 e itens vazios. |

A inserção SQL verifica o leitor num estado que o schema permite. A admissão sem lançamento pelo binário está bloqueada por padrão em `pending_identity/release_missing`.

## ESTADO

- Feito: método, script, rodada de 300 mil com duas fases, reprodução dos denominadores e da projeção autônoma.
- Pendente: rodada de 1 milhão, consolidação de planos e custo de escrita, propostas e registro no Backlog. AC1 depende do usuário.
