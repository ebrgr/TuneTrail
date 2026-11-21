---
id: ISSUE-082
title: 082 — Outbox durável no lugar do relay best-effort
status: Em andamento
assignee:
  - '@ebrgr'
created_date: '2026-11-14T00:39:00Z'
updated_date: '2026-11-21T11:12:00Z'
labels:
  - ingestao
  - api
milestone: m-11
dependencies:
  - ISSUE-081
references:
  - backlog/evidencias/issue-082/PLANO.md
  - backlog/evidencias/issue-082/MEDICOES-E2.md
  - backlog/evidencias/issue-082/VERIFICACAO-E3.md
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
- [x] #1 Relay vira destino da outbox. Na via legada/sombra e em playing_now, preserva URL, cabeçalhos e corpo byte a byte em relação à v0.47. Na via ativa, somente eventos aceitos da API ListenBrainz são elegíveis para entrega durável por evento em micro-lotes, com ACK independente da fila.
- [x] #2 Falhas e 429 são repetidas com backoff e Retry-After
- [x] #3 Encerramento drena ou persiste a fila sem perda
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

author: Codex
created: 2026-11-20T23:08:00Z

---

E1 concluída na base v0.48. Plano v3 em 4582311 registra condições para E2 e a decisão de semear a allowlist no upgrade. Transporte isolado em internal/relay no commit fe400da: resultado tipado, cabeçalhos de pausa, dialer SSRF e testes. Paridade legada confirmada com relay.go idêntico à v0.47 e golden via proxy de ambiente. Teste do pacote com race e make check passaram (13/13 E2E). Aguardando Reviewer antes da E2.

---

author: Codex
created: 2026-11-20T23:27:00Z

---

E1, retorno ao Reviewer: corrigidos os três itens do parecer NÃO APROVADO, em ddad548 (metadados), 191b279 (NAT64 local) e e7f98fd (token com controle). 651d01e cobre faixas reservadas e host:porta canônico. Plano v4 incorpora a semeadura restrita, as orientações para E2 e os limites DNS conhecidos. Teste de internal/relay com race e make check passaram, com 13/13 jornadas E2E. E2 aguarda nova revisão.

---

author: Codex
created: 2026-11-21T00:30:00Z

---

E2 concluída: 012 aditiva com política vigente legacy, guardas D4 efetivas, backup v11, semeadura e pausas transacionais. Store com intenções v9/occurrence, contador sem COUNT na escrita, claim em lote e CAS, encoders v1/v2 e testes de recuperação. Ensaio com 300 mil recibos: 012 em 1,43 s, CHECK isolado em 1,42 s. make check passou com 13/13 E2E, sem bloqueio de sandbox. Race direcionado passou sem a fixture de 100 mil itens, que excedeu o timeout instrumentado e passou na suíte normal. Handler, admissão durável e worker ficam para E3, após revisão de invariantes pelo Advisor Opus high.

---

author: Codex
created: 2026-11-21T01:47:00Z

---

Retorno da revisão E2 (Opus high, REPROVADO COM RESSALVAS): emendas aplicadas após aprovação das decisões a–e. Semeadura com DNS deduplicado e concorrência limitada, exceções apenas para administrador e bootstrap novo coberto. A 012 foi emendada antes da publicação para invalidar por geração. Em 100 mil itens, troca e remoção caíram de 532/514 ms para 174/125 µs. Renovação e liberação usam a chave por ID no EXPLAIN. Exclusão descarta abertos e preserva diagnóstico de sent sem chave de escuta. MEDICOES-E2 inclui 300 mil marcadores e PLANO v5 registra o corte da E3. make check passou (13/13 E2E), assim como os testes direcionados com race. Parada para revisão antes da E3.

---

author: Codex
created: 2026-11-21T02:14:00Z

---

E3 concluída. D1(a) emendada na 012 pré-publicação com agregados diários, retenção de 30 dias e purga total. Gate durável ativo no mesmo corte do handler e do worker. Replay byte a byte contra v0.48, pausas e bisseção, SIGTERM com HTTP em voo e três cenários SIGKILL verificados. SQLite cheio conserva a claim para recuperação. go test direcionado com race e make check passaram (13/13 E2E). Sem alteração de status. Parada para Reviewer, depois Advisor Opus high sobre encerramento e falhas ambíguas.

---

author: Codex
created: 2026-11-21T02:29:00Z

---

Retorno E3 à revisão: obrigatórios corrigidos em 954d174 (pausa após 2xx sem prazo do servidor) e c905e37 (espera após falha pré-POST, source_invalid e config_invalid). Ressalvas aplicadas, com decisão R5 registrada no plano, código e CHANGELOG. Testes dirigidos com race e make check passaram, com 13/13 E2E. Worktree permanece na E3 para nova revisão e posterior parecer do Advisor sobre encerramento e falhas ambíguas. AC4 segue aberto.

---

author: Codex
created: 2026-11-21T02:45:00Z

---

Decisão do usuário de 20/11/2026 aplicada: novas opções de runtime da 082 via Servidor root e conf.toml, sem nova variável de ambiente. Auditoria e constantes documentadas no PLANO. allow_http e allow_private_network são globais e desligadas por padrão, com reinício. Exceções por destino editáveis em formulários nativos com política root e CSRF. Host e porta configurados autorizados em avisos; DNS resolvido e segredos continuam excluídos. Diagnóstico de validação detalhado para root e somente classe para os demais. Jornada administrativa, testes dirigidos com race e make check passaram (13/13 E2E). Parada para Advisor sobre configuração, encerramento e falhas ambíguas.

---

author: Codex
created: 2026-11-21T11:12:00Z

---

Retorno pós-E3 aprovado pelo usuário em 21/11 aplicado. DNS de boot pausa por 30 s e tenta novamente; loopback e link-local exigem exceção por destino mesmo com a chave global; a UI e o guia descrevem o alcance para não-root e a via bruta. Diagnóstico público consolidado, autoridades IDNA normalizadas, aviso de CIDR amplo e pausa preventiva em INFO. SIGTERM interrompe claims e resolvedor de imediato; 2xx conhecido é persistido com contexto independente. make check passou (13/13 E2E), testes dirigidos com -race e graphify AST passaram. Cenário SIGTERM com HTTP em voo saiu em 4,005 s e retomou após reinício. Guia e CHANGELOG cobrem reentrega e impacto do upgrade. Retenção segue para ISSUE-083; medições e confirmação de deduplicação de Maloja/multi-scrobbler ficam na E4. Aguardando nova revisão do Advisor, sem iniciar E4 nem alterar status ou AC4.

---
<!-- COMMENTS:END -->
