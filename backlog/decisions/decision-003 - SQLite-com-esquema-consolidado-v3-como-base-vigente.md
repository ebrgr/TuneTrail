---
id: decision-003
title: SQLite com esquema consolidado v3 como base vigente
date: '2026-11-12T23:40:00Z'
status: accepted
---
## Context
Referências a esquemas anteriores e v4 apareciam em documentos históricos (doc-018).


## Decision
O esquema SQLite consolidado v3, com migrações Goose, é a base atual; consultar BANCO_DE_DADOS antes de mudar a persistência.


## Consequences
Novas tabelas (por exemplo ingestão e outbox do roadmap 2.0) exigem migração incremental, backup e testes de compatibilidade; migrações consolidadas não são alteradas retroativamente.

