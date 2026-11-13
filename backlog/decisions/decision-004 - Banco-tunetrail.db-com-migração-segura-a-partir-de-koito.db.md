---
id: decision-004
title: Banco tunetrail.db com migração segura a partir de koito.db
date: '2026-11-12T23:40:00Z'
status: accepted
---
## Context
A renomeação do projeto exigia trocar o nome do banco sem perder dados de instalações existentes (ISSUE-046).


## Decision
Na inicialização, criar backup consistente e migrar `koito.db` para `tunetrail.db`, preservando o esquema v3 e protegendo contra conflitos.


## Consequences
Ficam como lacunas registradas: testes sistemáticos de interrupção, falta de espaço e restauração completa (doc-017).

