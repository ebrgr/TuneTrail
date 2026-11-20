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

| Escutas totais | Perfil público | Perfil protegido | Outro perfil | Perfil vazio |
| ---: | ---: | ---: | ---: | ---: |
| 300.000 | 100.001 | 100.000 | 99.999 | 0 |
| 1.000.000 | 333.335 | 333.333 | 333.332 | 0 |

O SQLite que produziu os planos foi o 3.54.1. Os pares base/com índice usam a mesma máquina e volume. Os POSTs gravaram 103 escutas por fase, com 100 amostras cronometradas.

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
| 1.000.000 | `stats_year` | 297.40 / 303.17 / 318.58 | 246.65 / 270.30 / 287.67 |
| 1.000.000 | `stats_all` | 994.32 / 1015.69 / 1038.01 | 779.93 / 827.89 / 857.43 |
| 1.000.000 | `top_tracks` | 32.47 / 32.78 / 32.86 | 25.24 / 26.49 / 29.82 |
| 1.000.000 | `top_albums` | 43.03 / 44.20 / 44.72 | 28.38 / 29.59 / 33.80 |
| 1.000.000 | `top_artists` | 64.50 / 65.70 / 66.00 | 48.65 / 49.24 / 74.48 |
| 1.000.000 | `history_first` | 4.19 / 5.23 / 5.60 | 4.19 / 4.92 / 5.61 |
| 1.000.000 | `history_page_400` | 21.20 / 21.71 / 21.92 | 15.02 / 15.64 / 16.28 |
| 1.000.000 | `heatmap_data_recent` | 82.70 / 83.47 / 83.80 | 82.67 / 83.61 / 84.59 |
| 1.000.000 | `heatmap_data_year` | 99.60 / 99.96 / 100.11 | 99.69 / 119.11 / 173.78 |
| 1.000.000 | `rewind_api` | 1286.85 / 1329.69 / 1348.33 | 1201.79 / 1227.67 / 1247.17 |
| 1.000.000 | `rewind_html` | 1291.39 / 1311.41 / 1342.91 | 1206.60 / 1225.06 / 1298.19 |
| 1.000.000 | `home_all` | 2493.58 / 2606.89 / 2634.76 | 1981.22 / 2035.50 / 2143.60 |
| 1.000.000 | `home_week` | 1127.78 / 1213.69 / 1280.26 | 893.04 / 901.94 / 912.61 |

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
| 1.000.000 | `plays` | 0.94 / 0.96 / 0.97 | 0.65 / 0.66 / 0.66 |
| 1.000.000 | `minutes` | 15.36 / 15.69 / 16.06 | 7.43 / 7.54 / 7.55 |
| 1.000.000 | `unique_artists` | 32.13 / 32.43 / 37.37 | 22.76 / 22.87 / 22.94 |
| 1.000.000 | `active_days` | 56.88 / 59.55 / 74.31 | 57.41 / 58.26 / 58.37 |
| 1.000.000 | `longest_streak` | 9.23 / 9.99 / 10.39 | 9.84 / 10.41 / 11.04 |
| 1.000.000 | `activity` | 12.37 / 12.50 / 12.70 | 13.39 / 13.88 / 13.99 |
| 1.000.000 | `top_tracks` | 11.45 / 12.67 / 17.24 | 11.15 / 11.28 / 11.32 |
| 1.000.000 | `top_albums` | 22.29 / 23.05 / 23.36 | 13.85 / 13.93 / 14.02 |
| 1.000.000 | `top_artists` | 35.02 / 35.28 / 36.89 | 24.67 / 26.61 / 27.44 |
| 1.000.000 | `history_page_400` | 8.20 / 8.27 / 8.34 | 5.39 / 5.64 / 6.51 |
| 1.000.000 | `new_tracks` | 11.33 / 11.56 / 11.86 | 11.57 / 15.14 / 44.22 |
| 1.000.000 | `profile_counts` | 19.16 / 19.45 / 19.67 | 19.48 / 29.82 / 52.44 |
| 1.000.000 | `release_legacy` | 21.70 / 22.15 / 22.23 | 14.18 / 19.57 / 27.54 |
| 1.000.000 | `release_occurrence` | 21.47 / 21.77 / 21.80 | 13.56 / 15.68 / 17.13 |
| 1.000.000 | `recording_occurrence` | 23.82 / 24.52 / 42.80 | 16.04 / 21.19 / 34.58 |

### Ingestão e recursos

| Carga | Fase | POST p50 / p95 / p99 (ms) | DB (MB) | Pico RSS leitura / escrita (MiB) | Pico diretório leitura / escrita (MB) | Pico WAL leitura / escrita (MB) |
| ---: | --- | ---: | ---: | ---: | ---: | ---: |
| 300.000 | baseline | 66.00 / 72.25 / 105.80 | 201.83 | 77.88 / 76.81 | 339.51 / 343.76 | 0.00 / 4.13 |
| 300.000 | indexed | 65.26 / 66.15 / 68.23 | 207.25 | 74.97 / 73.56 | 344.94 / 349.25 | 0.00 / 4.19 |
| 1.000.000 | baseline | 199.99 / 235.84 / 255.10 | 662.24 | 73.52 / 68.27 | 1113.85 / 1118.12 | 0.00 / 4.14 |
| 1.000.000 | indexed | 196.99 / 212.76 / 221.23 | 680.09 | 57.83 / 56.16 | 1131.73 / 1136.00 | 0.00 / 4.15 |

## Planos de consulta

- **Intervalos e dias ativos.** As contagens por período usam busca de faixa por perfil e instante, coberta por índice. Dias ativos usam busca por perfil com teto de tempo. Nesse último caso, o teto abrange todo o histórico do perfil e produz uma linha por segundo distinto para o agrupamento em Go. A atividade e o streak também agregam inicialmente por segundo; o índice experimental preserva os planos desses caminhos. O custo cresce com a quantidade de segundos ativos mesmo sem varredura da tabela inteira.
- **Rankings.** Artistas e álbuns buscam escutas por perfil e instante e acessam créditos ou `tracks` por índice/chave primária. `GROUP BY` e `ORDER BY` usam B-trees temporárias, inclusive para `RANK()`, limite/paginação e ordenação final da projeção. Nos dois volumes, o top de faixas base usou `idx_listens_legacy_identity(profile_id=?)` e percorreu o índice de todo o perfil. O índice experimental restringiu por perfil e intervalo e introduziu B-tree para agrupar por faixa. O CTE ranqueado ainda é percorrido depois.
- **Novidades do Rewind e contagem global.** `new_tracks` busca por perfil sem restringir instante, pois a primeira escuta é calculada com MIN antes de filtrar o resultado. `profile_counts` é uma leitura diagnóstica de todos os perfis e faz `SCAN` integral de um índice de cobertura. Nenhum plano implica ler todos os perfis para um ranking individual.
- **Histórico paginado.** Base: busca por faixa no índice perfil/instante, junção de título/alias e B-tree para o último termo do `ORDER BY`. O índice experimental cobre a ordem atual por instante/faixa e elimina essa B-tree, mas o `OFFSET` e a contagem total permanecem. A consulta isolada da página 400 melhora, enquanto o HTTP inclui a contagem e o enriquecimento dos itens. A ISSUE-135 muda a chave de navegação para instante/evento. O índice de cursor `idx_listens_event_cursor` já existe no schema.
- **Ponte de identidade.** Álbuns, duração, histórico e alguns filtros usam `tracks` ou suas views. O comparativo isolado troca a junção de lançamentos pela PK de `occurrences` e confere igualdade da lista agrupada. O índice experimental cobre o lado de `listens` em ambas as consultas. A troca da PK de catálogo mantém as B-trees de agrupamento e ordenação. `recording_occurrence` agrupa pela gravação com cardinalidade diferente; a fixture da 133 cria uma gravação por faixa, portanto esse tempo não representa um catálogo reconciliado com várias edições por gravação.

Planos completos, percentis, recursos e contagens por perfil estão nas medições de [300 mil](MEDICOES-300000.json) e [1 milhão](MEDICOES-1000000.json).

## Resultado do índice experimental

Em 1 milhão de escutas, o p95 da Home em Tudo caiu de 2,607 s para 2,036 s (−22%) e o de Semana de 1,214 s para 0,902 s (−26%). A soma isolada de duração caiu de 15,69 para 7,54 ms no p95 e o ranking HTTP de álbuns de 44,20 para 29,59 ms. A troca da junção de `tracks` para `occurrences`, com os índices originais, teve medianas próximas de 22 ms e preservou os totais por lançamento nesta fixture. A maior diferença na consulta isolada veio da cobertura do índice.

A página 400 ganhou 6,07 ms no p95 HTTP com 1 milhão, enquanto em 300 mil ficou estável em cerca de 20 ms. O heatmap recente manteve plan e mediana similares. O p95 do Rewind HTML caiu de 1,311 para 1,225 s em 1 milhão. Mesmo com o índice, a Home em Tudo continuou acima de 2 s no p95. A seleção semanal também lê o painel histórico, conforme `internal/webui/home.go:27–33`.

O custo consolidado observado foi de +5,42 MB na base de 300 mil e +17,85 MB na de 1 milhão, incluindo as 103 escutas da primeira fase. O p50 dos POSTs legados ficou em 66,00 → 65,26 ms e 199,99 → 196,99 ms, respectivamente. O WAL amostrado na escrita cresceu de 4,13 para 4,19 MB e de 4,14 para 4,15 MB. Com a ordem fixa do ensaio, as amostras não estabelecem aumento ou redução causal da latência de ingestão. A expansão em disco é observável. O ensaio justifica retestar o índice de cobertura após a troca dos leitores da ISSUE-135, com a escrita por ocorrência e comparação em ordem alternada. Para a paginação por evento, medir primeiro o índice de cursor existente.

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

## Tarefas propostas para aprovação

| Título | Motivo verificado | Dependência |
| --- | --- | --- |
| Alinhar a média diária ao período consultado | O mesmo janeiro de 60 escutas produziu 0,50 em `/stats` e 1,94 em `/summary`. Aplicar a semântica aprovada e testar limites, zero, perfil e compatibilidade do JSON. | Aceite da ISSUE-092. Coordenar períodos parciais e exclusões com ISSUE-120 e as comparações com ISSUE-093. |
| Reduzir leituras históricas repetidas na Home e no Rewind | Em 1 milhão, p95 de 2,607 s na Home em Tudo, 1,214 s na Home semanal e 1,311 s no Rewind HTML. O índice reduziu parte do custo e ampliou o banco em 17,85 MB. Medir novamente depois do corte e isolar o custo do painel histórico, das novidades e das consultas repetidas antes de adotar otimizações. | ISSUE-135. Usar a nova medição ao planejar as barras da ISSUE-167. |

Na ISSUE-135, incluir no teste de paridade o caso comprovado de uma escuta contabilizada com histórico e top vazios. Cobrir também o balde sem lançamento, simultaneidade e cursor por evento. Esse trabalho já pertence à troca dos leitores. As medições não justificam uma tarefa isolada para indexar o heatmap ou o histórico legado.

## Limites do ensaio

Uma execução por volume, com índice sempre na segunda fase. Cache quente e clientes seriais. Os percentis descrevem as amostras desta máquina. A fase maior usa 40 leituras por operação e o p99 coincide com o máximo da amostra. Variações pequenas entre fases podem vir da ordem, do cache ou do sistema. Os perfis medidos têm distribuição uniforme com cerca de um terço do volume por perfil. Caudas e cardinalidades de históricos reais podem ser diferentes.

O disco consolidado inclui páginas livres e 103 escutas a mais na fase com índice. O tamanho do diretório inclui backups de migração. RSS é o pico amostrado do servidor durante as fases, sem contar o gerador, o cliente Python ou o cache de página do sistema. Arquivos temporários já desvinculados não aparecem na amostragem do diretório.

Fora do ensaio ficaram concorrência de leitura/escrita, cauda a frio, importação de arquivo em massa, latência do worker ativo e dos provedores, custo de criação/reconstrução de índice, cache ou materialização, agrupamento real de edições, comparação de YTD/DST em carga, análise de duração observada e a renderização no navegador. A ISSUE-134 mediu separadamente a admissão ativa e a visibilidade com 300 mil escutas num único perfil. Seu resultado usa uma fixture e protocolo diferentes deste A/B.

## ESTADO

- Feito: inventário e opções no doc-044, ensaios e planos de 300 mil e 1 milhão, comparação de ingestão legada antes/depois e propostas listadas para aprovação.
- Pendente: aprovação da semântica (AC1) e decisão sobre as tarefas propostas (AC3). O AC2 está verificado.
