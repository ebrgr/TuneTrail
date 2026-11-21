---
id: ISSUE-173
title: 173 — Política de retry e fila de falhas permanentes do resolvedor de ingestão
status: Done
assignee:
  - '@ebrgr'
created_date: '2026-11-20T18:20:00Z'
updated_date: '2026-11-21T21:59:00Z'
labels:
  - ingestao
milestone: m-11
dependencies:
  - ISSUE-081
references:
  - backlog/evidencias/issue-081/ENSAIO-C6.md
  - backlog/evidencias/issue-173/PLANO.md
type: feature
ordinal: 186000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: ressalva R-a do Reviewer e medição da C6 da ISSUE-081 (backlog/evidencias/issue-081/ENSAIO-C6.md, seção R-a). No modo ativo o resolvedor incrementa attempt_count mas nenhum código usa o contador: um recibo confirmado com 200 cuja resolução falha de forma persistente volta a retryable a cada minuto, para sempre e sem aparecer para o operador (medido: tentativas em 0,99 s, 60,99 s e 121,00 s; recuperou na 4ª tentativa quando a falha foi removida). Proposta medida para aprovar: até 5 tentativas automáticas para falhas transitórias, com espera de 1, 2, 4 e 8 minutos; na 5ª falha, reter recibo, chave, motivo e contagem numa fila durável de falhas permanentes, com alerta e reprocessamento explícito após a correção; encerrar com motivo registrado os erros de entrada reconhecidamente determinísticos; nunca descartar recibo confirmado (recuperável sempre). Decisões do usuário em aberto: teto e esperas, onde mostrar a fila (menu de saúde/admin), quem reprocessa (root) e se a fila de revisão (ISSUE-083) compartilha a mesma tela. Inclui R-b (decisão do usuário em 20/11/2026): erro genérico antes de o modo ser conhecido deixa de dar 500 e segue pelo caminho legado. Pré-requisito: ISSUE-081 publicada. Deve ser concluída antes de ativar o piloto da ingestão. Escopo a refinar com o Advisor (effort high: concorrência, migração) antes de implementar.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 O resolvedor limita as tentativas automáticas a 5, com esperas de 1, 2, 4 e 8 minutos, e move o recibo esgotado para uma fila durável de falhas permanentes sem perder chave, motivo ou contagem
- [x] #2 O dono do perfil e o root enxergam a fila na central de saúde e reprocessam um recibo de forma explícita e idempotente; o reprocessamento reaproveita a mesma resolução transacional
- [x] #3 Erros de entrada determinísticos são encerrados com motivo registrado; falhas transitórias continuam recuperáveis
- [x] #4 Ensaio sintético com falha persistente e transitória, crash e retomada por lease, sem escuta confirmada perdida; Advisor (effort high) registrado
- [x] #5 R-b: com o interruptor mestre ligado, um erro genérico (que não é de armazenamento) antes de o modo do perfil ser conhecido segue pelo caminho legado em vez de responder 500; erros de armazenamento e BUSY continuam em 503 com Retry-After; testado por classe de erro
- [x] #6 A central de saúde organiza as seções em duas ou três colunas no desktop e empilhadas no celular, compartilhada com a fila de revisão da ISSUE-181, com o visual e o hover do padrão do sistema
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-21T16:26:00Z

---

Decisões do usuário (21/11/2026): 5 tentativas com esperas de 1/2/4/8 min; fila como seção da central de saúde; reprocessam dono do perfil e root (AC #2 reescrito); mesma central com a fila de revisão da 181, em seções separadas, com 2 a 3 colunas no desktop e empilhadas no celular. Plano para o Advisor (effort high) em backlog/evidencias/issue-173/PLANO.md. Nenhum código escrito.

---

author: claude
created: 2026-11-21T16:28:00Z

---

Decisão do usuário (21/11/2026): estacionar com a opção A (colunas parked_at e park_reason, state continua retryable, sem reconstruir ingest_events). Despachado ao Codex pelo Orca. O parecer do Advisor (AC #4) continua pendente.

---

author: ebrgr
created: 2026-11-21T16:28:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-173` na branch `feat/issue-173-politica-de-retry-e-fila` criada a partir de `main`.

---

author: codex
created: 2026-11-21T16:43:00Z

---

Decisões de implementação: a migração 013 acrescenta colunas e índices sem reconstruir ingest_events. attempt_count continua acumulado e retry_attempt_base delimita o orçamento. Retomar lease vencido incrementa o total de claims e work_version, mas repete o mesmo número de tentativa no orçamento. Recibos v12 mantêm o histórico e, se já estavam no teto, estacionam na próxima falha. Estacionar e liberar lease são uma única atualização. O reprocessamento usa transação de escrita e CAS nas duas versões; recibos já reagendados ou de outro perfil resultam em no-op.

---

author: codex
created: 2026-11-21T16:46:00Z

---

Verificação: make check passou (13/13 jornadas E2E), make ui.reference.check conferiu 176 arquivos sem diferenças e os testes focados passaram com -race. Ensaios com trigger e relógio injetado cobriram falha persistente e transitória, esperas de 1/2/4/8 min, retomada após lease, recuperação de escutas após HTTP 200, JSON inválido e migração v12→v13 com backup v12. Testados reprocessamento concorrente e idempotente, dono/root, CSRF, isolamento e ausência de segredos, além das classes de erro R-b. EXPLAIN confirmou os índices parciais do claim e da seção. Chrome verificou layout 1/2/3 colunas, sem rolagem horizontal, com foco e hover em Yuu e Pearl. AC #4 segue pendente do parecer do Advisor. A central está preparada para a seção da 181.

---

author: claude
created: 2026-11-21T18:02:00Z

---

Reviewer: sem BUG; 3 RISCOs menores (ver backlog/evidencias/ISSUE-173/PARECER-REVIEWER.md). Testes -race OK. AC #4 pendente do Advisor.

---

author: advisor
created: 2026-11-21T21:20:00Z

---

Advisor (effort high): APROVADO com ressalvas não bloqueantes; ver backlog/evidencias/issue-173/PARECER-ADVISOR.md. Melhorias 1 e 2 (recibo que excede a lease nunca estaciona; interrupção longa estaciona a fila, sem reprocessar todos) ficam para follow-up.

---

author: ebrgr
created: 2026-11-21T21:58:00Z

---

Verificação: go test -race (sqlite, adminui, engine) OK; make check 13/13 E2E (Codex); Reviewer sem BUG; Advisor (effort high) APROVADO com ressalvas, follow-up ISSUE-182. Todos os ACs marcados.

---

author: ebrgr
created: 2026-11-21T21:59:00Z

---

Aceite do usuário em 21/11/2026: aceito e autorizado a publicar (sem push).

---
<!-- COMMENTS:END -->
