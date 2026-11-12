---
id: decision-005
title: Versionamento TuneTrail 0.x independente do Koito
date: '2026-11-12T23:40:00Z'
status: accepted
---
## Context
O fork evolui em ritmo próprio, mas reconhece a versão do Koito usada como base (ISS-044).


## Decision
Usar TuneTrail 0.x (`make version.bump` antes de cada commit; `make version.check`); 1.0 reservado para decisão posterior do usuário. O README cita o motor Koito 0.3.2.


## Consequences
Versões e tags de imagem seguem 0.x; a versão do Koito aparece apenas como referência de origem.

