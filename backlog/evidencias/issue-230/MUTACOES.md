# ISSUE-230 — Prova por mutação na árvore real

Cada alteração proibida foi aplicada sozinha, o teste `TestArchitecture` foi executado e a árvore foi restaurada (27/11/2026).

```
## 1 handlers importa sqlite: FALHOU (esperado)
   [only-engine-imports-sqlite] engine/handlers -> internal/db/sqlite: only engine, which assembles the program, imports internal/db/sqlite
   [web-without-sqlite] engine/handlers -> internal/db/sqlite: the web packages must not reach internal/db/sqlite, directly or through another package
## 2 export importa sqlite: FALHOU (esperado)
   [only-engine-imports-sqlite] internal/export -> internal/db/sqlite: only engine, which assembles the program, imports internal/db/sqlite
   [web-without-sqlite] engine/handlers -> internal/db/sqlite: the web packages must not reach internal/db/sqlite, directly or through another package
   [web-without-sqlite] internal/adminui -> internal/db/sqlite: the web packages must not reach internal/db/sqlite, directly or through another package
   [web-without-sqlite] internal/adminweb -> internal/db/sqlite: the web packages must not reach internal/db/sqlite, directly or through another package
   [web-without-sqlite] internal/webui -> internal/db/sqlite: the web packages must not reach internal/db/sqlite, directly or through another package
## 3 webui alcanca sqlite via relayoutbox: FALHOU (esperado)
   [web-without-sqlite] internal/webui -> internal/db/sqlite: the web packages must not reach internal/db/sqlite, directly or through another package
## 4 export com driver SQL: FALHOU (esperado)
   [sql-only-in-sqlite] internal/export -> database/sql: SQL statements and SQL drivers stay inside internal/db/sqlite
## 5 export com texto SQL: FALHOU (esperado)
   [sql-only-in-sqlite] internal/export -> SQL statement: SQL statements and SQL drivers stay inside internal/db/sqlite
## 6 models importa mbz: FALHOU (esperado)
   [core-without-adapters] internal/db -> internal/mbz: internal/ingest, internal/models and internal/db must not reach adapters or the web interface
   [core-without-adapters] internal/models -> internal/mbz: internal/ingest, internal/models and internal/db must not reach adapters or the web interface
## 7 catalog importa webui: FALHOU (esperado)
   [domain-without-interface] internal/catalog -> internal/webui: domain packages must not import adminui, webui or engine
## 8 sqlite importa importer: FALHOU (esperado)
   [sqlite-without-new-business-imports] internal/db/sqlite -> internal/importer: internal/db/sqlite must not gain imports of business packages
## 9 excecao obsoleta (summary sem database/sql): FALHOU (esperado)
   [sql-only-in-sqlite] internal/summary -> database/sql no longer happens: delete this entry from exceptions
## árvore restaurada: ok  	github.com/ebrgr/TuneTrail/internal/architecture	0.263s
mutações não detectadas: 0
```
