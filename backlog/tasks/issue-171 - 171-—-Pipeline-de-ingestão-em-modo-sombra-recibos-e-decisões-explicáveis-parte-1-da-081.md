---
id: ISSUE-171
title: >-
  171 — Pipeline de ingestão em modo sombra: recibos e decisões explicáveis
  (parte 1 da 081)
status: Done
assignee:
  - '@ebrgr'
created_date: '2026-11-19T16:18:00Z'
updated_date: '2026-11-19T17:22:00Z'
labels:
  - ingestao
milestone: m-11
dependencies:
  - ISSUE-080
references:
  - backlog/evidencias/issue-171/VERIFICACAO.md
  - backlog/evidencias/issue-171/PARECER-REVIEWER.md
documentation:
  - doc-043
priority: high
type: feature
ordinal: 184000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: divisão da ISSUE-081 em 2026-11-19, por decisão do usuário, a partir do estudo do Advisor (doc-043). Esta é a PARTE 1: grava o recibo e a decisão de deduplicação ao lado do caminho atual, SEM mudar nenhuma contagem de escutas, para medir a taxa real de Revisar no banco verdadeiro antes da ativação (ISSUE-081, parte 2). Fica antes da ISSUE-133 porque cabe nas guardas da v4: confirmado pelo usuário e pela leitura da migração 004 que a v4 aceita duplicate_decision same/different/review com event_id NULL (004:56 e :62; state só received, pending_identity, pending_duplicate e invalid; closed_at só para invalid; nenhum CHECK liga duplicate_decision a event_id). Consequência: linhas de sombra nunca ficam fechadas nem aceitas; o estado delas permanece received ou pending_*.

Escopo (commits ISSUE-171:): (1) internal/ingest/envelope.go: tipos, snapshot saneado, digest e precisão de tempo (second ou estimated); (2) internal/db/sqlite/ingest_receipts.go: ReceiveBatch em transação curta (BEGIN IMMEDIATE) com os índices únicos de idempotência da v4; (3) internal/dedup: Compare(policy, recibo, candidato) devolve Igual, Diferente ou Revisar com reason_code, evidência e rule_version; janela de ±2 s só seleciona candidatos, nunca prova identidade; (4) handler ListenBrainz em modo sombra: grava o recibo antes do caminho atual; o 200 só sai depois do commit do recibo e 503 em falha de disco; MusicBrainz e HTTP fora da trava; playing_now continua só em memória, sem recibo; (5) importadores em sombra: digest do arquivo mais ordinal, contador Pending e retomada por cursor (coordenar com a ISSUE-134); (6) relatório da taxa de Revisar sobre cópia de banco real, sem tocar o banco real.

Decisões adotadas (comentário do Advisor na ISSUE-081): mesma faixa duas vezes no mesmo segundo, pela API e sem ID externo, é registrada como Revisar (contrato §5); escuta sem listened_at é aceita com a hora de chegada marcada estimated; o modo Revisar nunca altera contagens nesta parte. Fora do escopo: criar evento ou escuta nova, outbox (082), tela de revisão (083), remover a escrita direta em listens, mexer nas guardas da v4 ou criar migração (se a parte 1 exigir migração, parar e registrar). O bug do mbid_mapping é a ISSUE-170.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 O recibo de cada escuta e a decisão (same, different ou review) com reason_code, evidência e rule_version são gravados em ingest_events sem alterar nenhuma contagem de escutas; o caminho atual continua sendo a fonte da verdade
- [x] #2 O 200 da API só sai depois do commit do recibo e falha de disco dá 503, sem perder o lote; playing_now continua só em memória
- [x] #3 Testes de concorrência (envios iguais de conexões distintas disputando a mesma chave) e de repetições legítimas, incluindo mesma faixa no mesmo segundo, fuso e horário de verão
- [x] #4 Replay reproduzível em cópia privada comprova que a regra classifica o histórico persistido sem alterar a origem ou as escutas legadas. Replay sintético comprova Revisar entre fontes a 0–2 s e em duas entradas no mesmo segundo. A amostra histórica não estima a taxa em produção nem cobre reenvios descartados e tráfego entre fontes; registrar resultados, limites e consulta versionada para medir antes da ativação.
- [x] #5 Confirmado em teste que linhas de sombra com duplicate_decision same e event_id NULL são aceitas pela v4 sem nova migração
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: ebrgr
created: 2026-11-19T16:18:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-171` na branch `feat/issue-171-pipeline-de-ingestao-em-modo` criada a partir de `main`.

---

author: codex
created: 2026-11-19T16:19:00Z

---

Decisão pendente: para itens sem listened_at, o recibo atômico usa a hora de chegada, enquanto o caminho legado calcula a hora por item. Usar a chegada também no legado pode mudar a contagem por faixa e segundo. Proposta e impacto registrados em VERIFICACAO.md. Nenhum critério marcado.

---

author: codex
created: 2026-11-19T16:33:00Z

---

Verificação: cinco critérios comprovados. Aprovado o uso da hora de chegada somente no recibo estimated. Os testes cobrem commit, disco cheio, reenvio, fail-open, playing_now, WAL, fuso/horário de verão e a guarda v4. Replay em cópia privada: 0 de 80 escutas encaminhadas para Revisar, com hashes da origem e conteúdo legado preservados. A tarefa segue Em andamento.

---

author: reviewer
created: 2026-11-19T16:40:00Z

---

Reviewer (leitura, sem repetir testes): REPROVADO com 1 item obrigatório. [BUG] PurgeAllData, DeleteListen e ApplyProfileDelete deixam os recibos de sombra (snapshot e chave de arquivo) depois da exclusão, sem purga, o que viola o doc-040 §4/§7. Ressalvas: linhas de sombra received/closed_at NULL entram em idx_ingest_queue (a 081 precisa filtrá-las); o UPDATE no reenvio sobrescreve a decisão original (different vira same); a comparação sem normalização subestima a taxa de Revisar; a trava depende da ordem do INSERT em integrations; importador grava um recibo por transação; snapshot sem limites; logs sem código de erro; AC4 (0/80) quase tautológico e replay não versionado. Modo sombra, 503 e fail-open, idempotência, perfil vindo do token e ausência de migração: OK. Advisor só na parte 2 (effort=high). Parecer em backlog/evidencias/issue-171/PARECER-REVIEWER.md.

---

author: codex
created: 2026-11-19T16:59:00Z

---

Verificação após o parecer: exclusões atômicas com cascata e novo recibo após reimportar, decisões originais preservadas em reenvios equivalentes, normalização shadow-v2, BEGIN IMMEDIATE e disputa em WAL para as três chaves, lotes de 200 com cauda parcial. AC4 reescrito e comprovado com replay versionado: cópia histórica 0/80 sem mudança na origem ou em listens; sintético 4/10, incluindo fontes cruzadas a 0–2 s e repetição no mesmo segundo. O histórico não estima a taxa em produção. Ressalvas 3, 6 e 7 registradas na ISSUE-081. A tarefa segue Em andamento.

---

author: codex
created: 2026-11-19T17:18:00Z

---

Verificação do ponto 1 da leitura de 287914c: ApplyProfileDelete apaga todos os recibos e somente integrações de sombra do perfil na transação do histórico. ReceiveBatch, PurgeAllData e ApplyProfileDelete compartilham a definição das origens e identidades de sombra. TestShadowDeletePaths confirma que uma configuração externa, inclusive com prefixo file:, sobrevive à exclusão do histórico e que o outro perfil mantém seus dados. As entradas obrigatórias 3, 6, 7 e a divergência de horário estimated foram confirmadas no comentário da ISSUE-081. A tarefa segue Em andamento.

---

author: claude
created: 2026-11-19T17:22:00Z

---

Aceite explícito do usuário em 2026-11-19. Pareceres: Reviewer (Opus, effort high) REPROVADO por 1 item obrigatório (recibos de sombra sobreviviam à exclusão), corrigido em 287914c junto com as ressalvas 1, 2, 4, 5 e 8; ponto adicional do Claude (ApplyProfileDelete apagava todas as integrations do perfil) corrigido em aaf7b60 e revisado por leitura do diff. Ressalvas 3, 6 e 7 e o limite dos recibos estimated registrados na ISSUE-081 como entrada obrigatória da ativação. O AC4 foi reescrito e cobre apenas o que o relatório prova (0 de 80 na cópia real; 4 de 10 no replay sintético com origens cruzadas): a taxa de Revisar em produção ainda precisa de algumas semanas de modo sombra antes de habilitar a parte 2. Advisor obrigatório (effort high) sobre o diff da ativação na ISSUE-081.

---
<!-- COMMENTS:END -->
