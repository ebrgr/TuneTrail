---
id: ISSUE-186
title: >-
  186 — Aceitar escuta sem álbum como Lançamento desconhecido na ingestão por
  ocorrência
status: Aberta
assignee: []
created_date: '2026-11-22T17:02:00Z'
labels:
  - ingestao
  - catalogo
milestone: m-11
dependencies: []
type: feature
ordinal: 199000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: ensaio da fase 1 da ISSUE-172 (backlog/evidencias/issue-172/ENSAIO.md) e decisão do usuário em 22/11/2026. O escritor legado grava escutas sem álbum; a ingestão por ocorrência as deixa em pending_identity / release_missing, sem ação de revisão e fora das estatísticas (205 de 2.250 requisições no ensaio). Decisão: aceitar sem álbum, conforme o contrato da ISSUE-132 (doc-039): ocorrência provisória sem lançamento (release_id nulo, origem unknown), ligada à gravação quando identificável, sem inventar álbum e sem atribuir a um álbum conhecido; a escuta conta no total e na gravação e aparece no balde 'Lançamento desconhecido' do ranking por lançamento (ISSUE-135), nunca em um álbum real; se o álbum surgir depois, revisão explícita pode reassociar. Pontos de projeto a resolver no código (documentar a escolha): (1) evitar uma ocorrência e uma gravação novas por escuta quando o mesmo título e os mesmos artistas já existem (reuso por alias exato e créditos, sem fusão por similaridade; ambiguidade continua pending_identity); (2) como o legado resolve hoje escutas sem álbum, para manter paridade de contagem; (3) classificação 'incompleto' no Inbox (ISSUE-184), exportação e reimportação (ISSUE-183) preservando a ocorrência sem lançamento; (4) intenção de relay segue o mesmo caminho das escutas aceitas; (5) escutas sem artista ou sem título continuam rejeitadas como hoje (400 nos dois caminhos). Pré-requisito do corte do legado (ISSUE-172, fase 2). Advisor (effort high) obrigatório antes do aceite, por tocar a identidade.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Escuta sem álbum é aceita pela ingestão por ocorrência como ocorrência sem lançamento (Lançamento desconhecido), conta no total e na gravação e nunca em um álbum real
- [ ] #2 Reenvios e escutas repetidas da mesma faixa sem álbum reutilizam a mesma ocorrência e gravação por alias e créditos exatos, sem criar uma por escuta; ambiguidade segue para pending_identity
- [ ] #3 Paridade de contagem com o escritor legado no lote do ensaio (as 205 escutas sem álbum passam a ser contadas); escutas sem artista ou título continuam rejeitadas
- [ ] #4 Inbox (incompleto), ranking por lançamento (balde), exportação e reimportação e relay tratam a ocorrência sem lançamento; testes por classe; make check passa; Advisor (effort high) registrado
<!-- AC:END -->
