# Planos do ranking de faixas

Medição local em 21/11/2026, macOS arm64, Go 1.26.3 e driver SQLite
`modernc.org/sqlite v1.49.1`. A fixture em memória executa as migrações de
produção e `ANALYZE`. São 600.004 escutas, 10.002 ocorrências, dois perfis e
dois lançamentos. O perfil consultado tem 80% das escutas geradas. As edições
geradas compartilham a identidade de gravação duas a duas.

```sh
CGO_ENABLED=0 go test ./internal/db/sqlite -run '^TestTrackRankingQueryPlans$' -v -count=1
```

O teste usa o SQL de produção com o mesmo escopo de perfil e limite de 100
linhas. Cada tempo é a média de três páginas completas, incluindo leitura dos
metadados e dos artistas. A janela parcial cobre os segundos 150000 a 250000
da fixture.

| Filtro | Por gravação | Por lançamento |
| --- | ---: | ---: |
| Histórico completo | 68.8 ms | 193.0 ms |
| Janela parcial | 35.5 ms | 79.8 ms |
| Álbum na janela parcial | 31.0 ms | 36.0 ms |
| Artista na janela parcial | 31.3 ms | 36.5 ms |

Os oito planos consultaram as escutas com o índice de cobertura
`idx_listens_legacy_identity` por perfil, ocorrência e intervalo. O filtro por
álbum usou `idx_occurrences_release` e o filtro por artista usou a chave
primária de `artist_tracks`. Os aliases são consultados por chave depois da
paginação. O agrupamento por lançamento usou uma B-tree temporária e ficou
mais caro no histórico completo. Os índices existentes atendem aos acessos
observados na fixture. Os tempos em memória variam com volume, seletividade,
cache e armazenamento.

## Saída dos oito planos

### all-time/recording

```text
CO-ROUTINE RankedTracks
CO-ROUTINE (subquery-7)
CO-ROUTINE (subquery-8)
CO-ROUTINE TrackCounts
SCAN o USING COVERING INDEX idx_occurrences_recording
SEARCH main.listens USING COVERING INDEX idx_listens_legacy_identity (profile_id=? AND track_id=? AND listened_at>? AND listened_at<?)
SCAN TrackCounts
SCAN (subquery-8)
USE TEMP B-TREE FOR ORDER BY
SCAN (subquery-7)
USE TEMP B-TREE FOR ORDER BY
SCAN r
SEARCH o USING INTEGER PRIMARY KEY (rowid=?)
SEARCH rec USING INTEGER PRIMARY KEY (rowid=?)
SEARCH rls USING INTEGER PRIMARY KEY (rowid=?) LEFT-JOIN
CORRELATED SCALAR SUBQUERY 4
SEARCH track_aliases USING INDEX sqlite_autoindex_track_aliases_1 (track_id=?)
CORRELATED SCALAR SUBQUERY 5
SEARCH release_aliases USING INDEX sqlite_autoindex_release_aliases_1 (release_id=?)
USE TEMP B-TREE FOR ORDER BY
```

### all-time/release

```text
CO-ROUTINE RankedTracks
CO-ROUTINE (subquery-7)
CO-ROUTINE (subquery-8)
CO-ROUTINE TrackCounts
SCAN o USING INDEX idx_occurrences_recording
SEARCH main.listens USING COVERING INDEX idx_listens_legacy_identity (profile_id=? AND track_id=? AND listened_at>? AND listened_at<?)
USE TEMP B-TREE FOR GROUP BY
SCAN TrackCounts
SCAN (subquery-8)
USE TEMP B-TREE FOR ORDER BY
SCAN (subquery-7)
USE TEMP B-TREE FOR ORDER BY
SCAN r
SEARCH o USING INTEGER PRIMARY KEY (rowid=?)
SEARCH rec USING INTEGER PRIMARY KEY (rowid=?)
SEARCH rls USING INTEGER PRIMARY KEY (rowid=?) LEFT-JOIN
CORRELATED SCALAR SUBQUERY 4
SEARCH track_aliases USING INDEX sqlite_autoindex_track_aliases_1 (track_id=?)
CORRELATED SCALAR SUBQUERY 5
SEARCH release_aliases USING INDEX sqlite_autoindex_release_aliases_1 (release_id=?)
USE TEMP B-TREE FOR ORDER BY
```

### period/recording

```text
CO-ROUTINE RankedTracks
CO-ROUTINE (subquery-7)
CO-ROUTINE (subquery-8)
CO-ROUTINE TrackCounts
SCAN o USING COVERING INDEX idx_occurrences_recording
SEARCH main.listens USING COVERING INDEX idx_listens_legacy_identity (profile_id=? AND track_id=? AND listened_at>? AND listened_at<?)
SCAN TrackCounts
SCAN (subquery-8)
USE TEMP B-TREE FOR ORDER BY
SCAN (subquery-7)
USE TEMP B-TREE FOR ORDER BY
SCAN r
SEARCH o USING INTEGER PRIMARY KEY (rowid=?)
SEARCH rec USING INTEGER PRIMARY KEY (rowid=?)
SEARCH rls USING INTEGER PRIMARY KEY (rowid=?) LEFT-JOIN
CORRELATED SCALAR SUBQUERY 4
SEARCH track_aliases USING INDEX sqlite_autoindex_track_aliases_1 (track_id=?)
CORRELATED SCALAR SUBQUERY 5
SEARCH release_aliases USING INDEX sqlite_autoindex_release_aliases_1 (release_id=?)
USE TEMP B-TREE FOR ORDER BY
```

### period/release

```text
CO-ROUTINE RankedTracks
CO-ROUTINE (subquery-7)
CO-ROUTINE (subquery-8)
CO-ROUTINE TrackCounts
SCAN o USING INDEX idx_occurrences_recording
SEARCH main.listens USING COVERING INDEX idx_listens_legacy_identity (profile_id=? AND track_id=? AND listened_at>? AND listened_at<?)
USE TEMP B-TREE FOR GROUP BY
SCAN TrackCounts
SCAN (subquery-8)
USE TEMP B-TREE FOR ORDER BY
SCAN (subquery-7)
USE TEMP B-TREE FOR ORDER BY
SCAN r
SEARCH o USING INTEGER PRIMARY KEY (rowid=?)
SEARCH rec USING INTEGER PRIMARY KEY (rowid=?)
SEARCH rls USING INTEGER PRIMARY KEY (rowid=?) LEFT-JOIN
CORRELATED SCALAR SUBQUERY 4
SEARCH track_aliases USING INDEX sqlite_autoindex_track_aliases_1 (track_id=?)
CORRELATED SCALAR SUBQUERY 5
SEARCH release_aliases USING INDEX sqlite_autoindex_release_aliases_1 (release_id=?)
USE TEMP B-TREE FOR ORDER BY
```

### album/recording

```text
CO-ROUTINE RankedTracks
CO-ROUTINE (subquery-7)
CO-ROUTINE (subquery-8)
CO-ROUTINE TrackCounts
SEARCH o USING INDEX idx_occurrences_release (release_id=?)
SEARCH main.listens USING COVERING INDEX idx_listens_legacy_identity (profile_id=? AND track_id=? AND listened_at>? AND listened_at<?)
USE TEMP B-TREE FOR GROUP BY
SCAN TrackCounts
SCAN (subquery-8)
USE TEMP B-TREE FOR ORDER BY
SCAN (subquery-7)
USE TEMP B-TREE FOR ORDER BY
SCAN r
SEARCH o USING INTEGER PRIMARY KEY (rowid=?)
SEARCH rec USING INTEGER PRIMARY KEY (rowid=?)
SEARCH rls USING INTEGER PRIMARY KEY (rowid=?) LEFT-JOIN
CORRELATED SCALAR SUBQUERY 4
SEARCH track_aliases USING INDEX sqlite_autoindex_track_aliases_1 (track_id=?)
CORRELATED SCALAR SUBQUERY 5
SEARCH release_aliases USING INDEX sqlite_autoindex_release_aliases_1 (release_id=?)
USE TEMP B-TREE FOR ORDER BY
```

### album/release

```text
CO-ROUTINE RankedTracks
CO-ROUTINE (subquery-7)
CO-ROUTINE (subquery-8)
CO-ROUTINE TrackCounts
SEARCH o USING INDEX idx_occurrences_release (release_id=?)
SEARCH main.listens USING COVERING INDEX idx_listens_legacy_identity (profile_id=? AND track_id=? AND listened_at>? AND listened_at<?)
USE TEMP B-TREE FOR GROUP BY
SCAN TrackCounts
SCAN (subquery-8)
USE TEMP B-TREE FOR ORDER BY
SCAN (subquery-7)
USE TEMP B-TREE FOR ORDER BY
SCAN r
SEARCH o USING INTEGER PRIMARY KEY (rowid=?)
SEARCH rec USING INTEGER PRIMARY KEY (rowid=?)
SEARCH rls USING INTEGER PRIMARY KEY (rowid=?) LEFT-JOIN
CORRELATED SCALAR SUBQUERY 4
SEARCH track_aliases USING INDEX sqlite_autoindex_track_aliases_1 (track_id=?)
CORRELATED SCALAR SUBQUERY 5
SEARCH release_aliases USING INDEX sqlite_autoindex_release_aliases_1 (release_id=?)
USE TEMP B-TREE FOR ORDER BY
```

### artist/recording

```text
CO-ROUTINE RankedTracks
CO-ROUTINE (subquery-7)
CO-ROUTINE (subquery-8)
CO-ROUTINE TrackCounts
SEARCH at USING COVERING INDEX sqlite_autoindex_artist_tracks_1 (artist_id=?)
SEARCH o USING INTEGER PRIMARY KEY (rowid=?)
SEARCH main.listens USING COVERING INDEX idx_listens_legacy_identity (profile_id=? AND track_id=? AND listened_at>? AND listened_at<?)
USE TEMP B-TREE FOR GROUP BY
SCAN TrackCounts
SCAN (subquery-8)
USE TEMP B-TREE FOR ORDER BY
SCAN (subquery-7)
USE TEMP B-TREE FOR ORDER BY
SCAN r
SEARCH o USING INTEGER PRIMARY KEY (rowid=?)
SEARCH rec USING INTEGER PRIMARY KEY (rowid=?)
SEARCH rls USING INTEGER PRIMARY KEY (rowid=?) LEFT-JOIN
CORRELATED SCALAR SUBQUERY 4
SEARCH track_aliases USING INDEX sqlite_autoindex_track_aliases_1 (track_id=?)
CORRELATED SCALAR SUBQUERY 5
SEARCH release_aliases USING INDEX sqlite_autoindex_release_aliases_1 (release_id=?)
USE TEMP B-TREE FOR ORDER BY
```

### artist/release

```text
CO-ROUTINE RankedTracks
CO-ROUTINE (subquery-7)
CO-ROUTINE (subquery-8)
CO-ROUTINE TrackCounts
SEARCH at USING COVERING INDEX sqlite_autoindex_artist_tracks_1 (artist_id=?)
SEARCH o USING INTEGER PRIMARY KEY (rowid=?)
SEARCH main.listens USING COVERING INDEX idx_listens_legacy_identity (profile_id=? AND track_id=? AND listened_at>? AND listened_at<?)
USE TEMP B-TREE FOR GROUP BY
SCAN TrackCounts
SCAN (subquery-8)
USE TEMP B-TREE FOR ORDER BY
SCAN (subquery-7)
USE TEMP B-TREE FOR ORDER BY
SCAN r
SEARCH o USING INTEGER PRIMARY KEY (rowid=?)
SEARCH rec USING INTEGER PRIMARY KEY (rowid=?)
SEARCH rls USING INTEGER PRIMARY KEY (rowid=?) LEFT-JOIN
CORRELATED SCALAR SUBQUERY 4
SEARCH track_aliases USING INDEX sqlite_autoindex_track_aliases_1 (track_id=?)
CORRELATED SCALAR SUBQUERY 5
SEARCH release_aliases USING INDEX sqlite_autoindex_release_aliases_1 (release_id=?)
USE TEMP B-TREE FOR ORDER BY
```
