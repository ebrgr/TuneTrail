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

- Feito: método, ensaio HTTP preliminar, reprodução dos denominadores e da projeção autônoma. Script versionado.
- Pendente: resultados de 300 mil e 1 milhão, análise de planos e custo de escrita, propostas e registro no Backlog. AC1 depende do usuário.
