---
id: ISSUE-184
title: >-
  184 — Central de saúde com Inbox (recebido por API key e importações) e Outbox
  (relay)
status: Done
assignee:
  - '@ebrgr'
created_date: '2026-11-21T23:31:00Z'
updated_date: '2026-11-22T15:32:00Z'
labels:
  - ingestao
  - ui
milestone: m-11
dependencies:
  - ISSUE-181
references:
  - backlog/evidencias/issue-184/VERIFICACAO.md
type: feature
ordinal: 197000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Pedido do usuário em 21/11/2026. A central de saúde (/admin/health) passa a ter duas tabelas lado a lado: à esquerda 'Inbox' (o que é recebido de outros serviços, como o Navidrome, e das importações) e à direita 'Outbox' (o que vai para o relay). Para enviar uma música, o Navidrome usa uma API key fornecida pelo TuneTrail; essa chave é a informação usada para filtrar o Inbox. Estado atual (lido no código): o recibo em ingest_events não guarda a chave (o lote do ListenBrainz sai com source_instance_id fixo 'listenbrainz'); a tabela api_keys tem label e profile_id; a central mostra só última entrega por fonte/cliente, decisões, fila do relay e falhas permanentes. Escopo: (1) migração aditiva (schema v17 após a 181, que usa v16) com referência à chave no recibo (por exemplo api_key_id) preenchida no handler de submissão; recibos antigos ficam como 'chave desconhecida'; nunca gravar nem exibir a chave bruta, só label e identificador interno; (2) Inbox: por chave (label) e por cliente, total recebido, aceitos, duplicados, inválidos e pendentes, em janelas de 24 h, 7 dias e 30 dias, mais as importações (arquivo) como origem própria; (3) metadados incompletos por chave: contagem de escutas com campos ausentes, em duas faixas ('faltante' = artista ou faixa ausente, que bloqueia a identidade; 'incompleto' = álbum, MBID ou duração ausentes), com um recibo de exemplo por tipo, só campos seguros do snapshot; (4) Outbox: reaproveitar a fila do relay, entregas, pausa e falhas já existentes na central, reorganizadas na coluna da direita; (5) visual e gate da central (dono do perfil e root), 2 colunas no desktop e empilhadas no celular com só min-width 40/64/90rem, hover igual ao padrão; (6) medir o custo das consultas com volume realista. Nota: o usuário escreveu 'coluna da esquerda' duas vezes; interpretado como esquerda = Inbox e direita = Outbox. Depende da ISSUE-181 (mesma tela e schema v16).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Recibos novos guardam a referência da API key (sem a chave bruta); recibos antigos aparecem como chave desconhecida
- [x] #2 Central de saúde com duas tabelas lado a lado: Inbox à esquerda (por API key, cliente e importações) e Outbox à direita (relay), empilhadas no celular
- [x] #3 Inbox mostra total, aceitos, duplicados, inválidos e pendentes por chave e janela de tempo, sem expor segredos
- [x] #4 Metadados faltantes e incompletos listados por chave, com exemplo seguro por tipo
- [x] #5 Isolamento por perfil e acesso só do dono e do root, com testes; consultas medidas com volume realista
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: ebrgr
created: 2026-11-22T15:06:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-184` na branch `feat/issue-184-central-de-saude-com-inbox` criada a partir de `main`.

---

author: codex
created: 2026-11-22T15:17:00Z

---

Verificação: Inbox por chave e cliente, importações e metadados; Outbox de relay com revisão e reprocessamento no Inbox. Contadores abaixo de 9 ms nas três janelas com 300 mil recibos. Migração medida em 7,00 s durante o startup. Make check, testes direcionados com detector de corridas e Orca em 390, 1100 e 1500 px passaram. Evidência em backlog/evidencias/issue-184/VERIFICACAO.md.

---

author: claude
created: 2026-11-22T15:23:00Z

---

Reviewer: sem BUG; 4 RISCOs (migração v19 com 7,0 s em 300 mil recibos e reconstrução de api_keys conferida; write amplification dos gatilhos do resumo diário; classificação de metadados acoplada às tags do snapshot; exemplo exibe título/artista ao dono e root). Testes sqlite -race, engine, handlers, middleware, adminui e webui OK. Ver backlog/evidencias/issue-184/PARECER-REVIEWER.md. Advisor (effort high) recomendado pela migração e exposição de dados; pendente a pedido do usuário.

---

author: ebrgr
created: 2026-11-22T15:32:00Z

---

Verificação: make check 13/13 E2E (Codex); go test -race sqlite/engine/adminui/webui OK; Reviewer sem BUG; consultas 6,3 a 8,3 ms com 300 mil recibos; tela conferida em 390/1100/1500 px; Advisor dispensado pelo usuário. ACs 1 a 5 marcados. Riscos aceitos: migração v19 de 7,0 s, escritas extras do resumo diário, classificação acoplada às tags do snapshot.

---

author: ebrgr
created: 2026-11-22T15:32:00Z

---

Aceite do usuário em 22/11/2026: aceito sem Advisor e autorizado a publicar (sem push). Riscos aceitos e registrados: migração v19 leva 7,0 s no startup com 300 mil recibos; gatilhos do resumo diário acrescentam escritas na admissão (remedir o p95 de ACK na ISSUE-172); a classificação de metadados depende das tags JSON do snapshot.

---
<!-- COMMENTS:END -->
