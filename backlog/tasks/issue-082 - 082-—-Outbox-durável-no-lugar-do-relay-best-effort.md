---
id: ISSUE-082
title: 082 — Outbox durável no lugar do relay best-effort
status: Em andamento
assignee:
  - '@ebrgr'
created_date: '2026-11-14T00:39:00Z'
updated_date: '2026-11-20T22:39:00Z'
labels:
  - ingestao
  - api
milestone: m-11
dependencies:
  - ISSUE-081
references:
  - backlog/evidencias/issue-082/PLANO.md
priority: high
type: feature
ordinal: 82000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: doc-010 §4.2 e doc-017 (ISSUE-020/034). Worker com backoff exponencial e jitter, `Retry-After`, repetição de 429 e drenagem no encerramento; chave de idempotência única. Revisão obrigatória do Advisor antes de implementar (AGENTS.md: mensageria/concorrência/criptografia → @Opus effort=high no Claude, @Sol effort=medium no Codex).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Relay vira destino da outbox. Na via legada/sombra e em playing_now, preserva URL, cabeçalhos e corpo byte a byte em relação à v0.47. Na via ativa, somente eventos aceitos da API ListenBrainz são elegíveis para entrega durável por evento em micro-lotes, com ACK independente da fila.
- [ ] #2 Falhas e 429 são repetidas com backoff e Retry-After
- [ ] #3 Encerramento drena ou persiste a fila sem perda
- [ ] #4 Parecer do Advisor registrado
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: ebrgr
created: 2026-11-20T22:04:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-082` na branch `feat/issue-082-outbox-duravel-no-lugar-do` criada a partir de `main`.

---

author: Codex
created: 2026-11-20T22:08:00Z

---

Fase 1: plano da outbox durável documentado, com mapa da v0.47, contrato de elegibilidade e payload, retry coordenado com a 173, migração 012, corte D4 e verificação por AC. Aguardando parecer obrigatório do Advisor (Opus high) e decisões sobre payload ativo, tentativas/idade, histórico e retenção antes da implementação.

---

author: advisor
created: 2026-11-20T22:39:00Z

---

Parecer do Advisor (Opus, esforço high), transmitido pelo usuário em 20/11/2026: REPROVADO COM RESSALVAS. Emendas exigidas: ACK independente da fila, micro-lotes com isolamento por bisseção, pausa por destino, payload determinístico a partir do snapshot, shutdown por persistência e lease, 2xx independente do corpo, política durável na admissão, proteção SSRF, eliminação da barreira de ACK e tratamento da troca de destino na transação. Revisão curta do contrato emendado antes da E1.

---

author: Codex
created: 2026-11-20T22:39:00Z

---

E0: PLANO.md v2 e AC1 reescrito. Decisões do usuário (20/11/2026): D1 paridade byte a byte na via legada/sombra e playing_now, entrega por evento na ativa. D2 política durável obrigatória em novas admissões ativas. D3 payload determinístico sem alterar Snapshot/Encode, tabela de campos para validação final. D4 micro-lotes de até 100. D5 30 s, fator 2, full jitter, 1 h, oito tentativas por item e 30 dias, helper comum à 173 com parâmetros separados. D6 429 longo/401/403/3xx pausam o destino. D7 URL alterada gera falha reprocessável, troca só de token preserva linhas. D8 HTTP/relay privado somente pela lista do root, demais usuários veem classe HTTP. D9 pré-082 fechado, perda aceita dos envios brutos interrompidos pelo upgrade. D10 teto de 100 mil itens abertos elegíveis por destino sem impacto no ACK. D11 falhas retidas até ação do operador, texto e visibilidade com a 083. Próxima etapa condicionada à revisão curta do Advisor.

---
<!-- COMMENTS:END -->
