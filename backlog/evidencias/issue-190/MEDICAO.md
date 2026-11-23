# ISSUE-190 — Medição de tempo ouvido

SQLite 3.54.1 via Python, schema de origem 22. A cópia temporária
fora da worktree tinha 80 escutas. Foram geradas mais 300.000 escutas com
a distribuição de faixas de um perfil da cópia. Após `ANALYZE`, as consultas
foram executadas com o filtro de perfil da aplicação. As consultas por período
abrangem a janela das 300.000 linhas geradas, com uma amostra adicional de
1.000 linhas para o total do período. Os tempos são medianas de
15 execuções com cache aquecido, em milissegundos.

## Migração

O `ALTER TABLE` da 023 demorou 84.70 ms sobre 300.080 escutas.
O banco passou de 26.641 para 26.641 páginas e o WAL recebeu 8.272
bytes desde o checkpoint. O SQLite valida o `CHECK` nas linhas existentes.
O tamanho do WAL e a contagem de páginas mostram o custo de uma alteração
de schema sem reescrita de `listens` ou dos índices. A medição isola o DDL.
O backup e a verificação de integridade do bootstrap têm custo próprio e
foram exercitados pelo teste de migração.

## Consultas

A primeira comparação manteve a coluna nula em todas as linhas e confirmou
os mesmos resultados. A última coluna mede a consulta com 150.000 linhas
preenchidas com metade da duração de catálogo.

| Consulta | Antes (ms) | Depois, NULL (ms) | Depois, com medições (ms) |
| --- | ---: | ---: | ---: |
| Faixa (detalhe) | 3.413 | 3.361 | 2.974 |
| Álbum (detalhe) | 19.417 | 16.419 | 13.871 |
| Artista (detalhe) | 18.966 | 16.801 | 13.992 |
| Período / Rewind | 204.486 | 20.021 | 19.453 |
| Período com 1.000 escutas | 0.055 | 0.065 | 0.065 |
| Artista no período | 15.986 | 17.742 | 14.206 |
| Álbum no período | 16.410 | 17.472 | 13.654 |
| Faixa no período | 3.041 | 3.374 | 2.903 |

Na janela ampla, a busca por faixa na tabela de escutas deu lugar a uma
varredura sequencial com consulta da faixa por chave primária. Na janela
de 1.000 escutas, o índice de perfil e data continuou a ser usado. Os
recortes por artista, álbum e faixa ficaram entre 0,33 e 1,76 ms mais caros
com a coluna toda nula nesta amostra.

### EXPLAIN QUERY PLAN

**Faixa (detalhe)**

- Antes — `SEARCH t USING INTEGER PRIMARY KEY (rowid=?) → SEARCH main.listens USING INDEX idx_listens_track_id_listened_at (track_id=?)`
- Depois — `SEARCH main.listens USING INDEX idx_listens_track_id (track_id=?) → SEARCH t USING INTEGER PRIMARY KEY (rowid=?) LEFT-JOIN`

**Álbum (detalhe)**

- Antes — `SEARCH t USING INDEX idx_tracks_release_id (release_id=?) → SEARCH main.listens USING INDEX idx_listens_track_id_listened_at (track_id=?)`
- Depois — `SEARCH t USING INDEX idx_tracks_release_id (release_id=?) → SEARCH main.listens USING INDEX idx_listens_track_id (track_id=?)`

**Artista (detalhe)**

- Antes — `SEARCH at2 USING COVERING INDEX sqlite_autoindex_artist_tracks_1 (artist_id=?) → SEARCH t USING INTEGER PRIMARY KEY (rowid=?) → SEARCH main.listens USING INDEX idx_listens_track_id_listened_at (track_id=?)`
- Depois — `SEARCH at2 USING COVERING INDEX sqlite_autoindex_artist_tracks_1 (artist_id=?) → SEARCH t USING INTEGER PRIMARY KEY (rowid=?) → SEARCH main.listens USING INDEX idx_listens_track_id (track_id=?)`

**Período / Rewind**

- Antes — `SCAN t → SEARCH main.listens USING INDEX idx_listens_track_id_listened_at (track_id=? AND listened_at>? AND listened_at<?)`
- Depois — `SCAN main.listens → SEARCH t USING INTEGER PRIMARY KEY (rowid=?) LEFT-JOIN`

**Período com 1.000 escutas**

- Antes — `SEARCH main.listens USING INDEX idx_listens_event_cursor (profile_id=? AND listened_at>? AND listened_at<?) → SEARCH t USING INTEGER PRIMARY KEY (rowid=?)`
- Depois — `SEARCH main.listens USING INDEX idx_listens_event_cursor (profile_id=? AND listened_at>? AND listened_at<?) → SEARCH t USING INTEGER PRIMARY KEY (rowid=?) LEFT-JOIN`

**Artista no período**

- Antes — `SEARCH at2 USING COVERING INDEX sqlite_autoindex_artist_tracks_1 (artist_id=?) → SEARCH t USING INTEGER PRIMARY KEY (rowid=?) → SEARCH main.listens USING INDEX idx_listens_track_id_listened_at (track_id=? AND listened_at>? AND listened_at<?)`
- Depois — `SEARCH at2 USING COVERING INDEX sqlite_autoindex_artist_tracks_1 (artist_id=?) → SEARCH t USING INTEGER PRIMARY KEY (rowid=?) → SEARCH main.listens USING INDEX idx_listens_track_id (track_id=?)`

**Álbum no período**

- Antes — `SEARCH t USING INDEX idx_tracks_release_id (release_id=?) → SEARCH main.listens USING INDEX idx_listens_track_id_listened_at (track_id=? AND listened_at>? AND listened_at<?)`
- Depois — `SEARCH t USING INDEX idx_tracks_release_id (release_id=?) → SEARCH main.listens USING INDEX idx_listens_track_id_listened_at (track_id=? AND listened_at>? AND listened_at<?)`

**Faixa no período**

- Antes — `SEARCH t USING INTEGER PRIMARY KEY (rowid=?) → SEARCH main.listens USING INDEX idx_listens_track_id_listened_at (track_id=? AND listened_at>? AND listened_at<?)`
- Depois — `SEARCH main.listens USING INDEX idx_listens_track_id_listened_at (track_id=? AND listened_at>? AND listened_at<?) → SEARCH t USING INTEGER PRIMARY KEY (rowid=?) LEFT-JOIN`
