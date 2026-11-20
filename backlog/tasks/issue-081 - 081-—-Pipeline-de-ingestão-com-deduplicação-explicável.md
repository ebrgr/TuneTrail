---
id: ISSUE-081
title: '081 — Pipeline de ingestão com deduplicação explicável: ativação (parte 2)'
status: Em andamento
assignee:
  - '@ebrgr'
created_date: '2026-11-14T00:39:00Z'
updated_date: '2026-11-20T17:57:00Z'
labels:
  - ingestao
milestone: m-11
dependencies:
  - ISSUE-171
  - ISSUE-133
references:
  - backlog/evidencias/issue-081/PLANO.md
  - backlog/evidencias/issue-081/RESSALVAS-ADVISOR.md
documentation:
  - doc-039
  - doc-043
priority: high
type: feature
ordinal: 81000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: doc-010 §4.2 e estudo do Advisor (doc-043). Dividida em 2026-11-19: a PARTE 1 (recibos e decisões em modo sombra, sem mudar contagens) é a ISSUE-171; esta é a PARTE 2, a ATIVAÇÃO, que vem depois da ISSUE-171 e da ISSUE-133 (modelo por ocorrência e remoção das guardas da v4, numa migração v7 ou posterior). Os dependentes originais da 081 (082, 084, 085, 089, 091, 134) continuam dependendo desta tarefa, porque precisam do pipeline ativo.

Escopo: internal/ingest (normalizar, gravar recibo, resolver, deduplicar, enfileirar) e internal/dedup passam a decidir de fato. Fluxo: validar, [transação] gravar o recibo, 200 igual a guardado com segurança (503 se falhar), resolver identidade, dedup Igual (liga o recibo ao evento existente, sem contar de novo), Diferente (cria evento ou escuta e a intenção na outbox da ISSUE-082, na mesma transação) ou Revisar (pendente, não conta até alguém revisar). Remove a escrita direta em listens. Deduplicação dentro do perfil; caso duvidoso vai para revisão e nunca é apagado automaticamente; com ID externo confiável e mesmo conteúdo, Igual automático. Decisões adotadas: o modo Revisar fica atrás de uma flag por perfil, DESLIGADA até a central de saúde da ISSUE-083 existir (senão as pendências ficam invisíveis); mesma faixa duas vezes no mesmo segundo pela API sem ID externo vai para Revisar; escuta sem listened_at é aceita como estimated. Riscos centrais: contagem duplicada na transição (usar os resultados da modo sombra da 171 antes de ligar), perda de escutas (200 só após commit), concorrência (resolution_version), reversão (backup VACUUM INTO; o binário antigo recusa o schema novo). Revisão obrigatória do Advisor (AGENTS.md: mensageria e concorrência, @Opus effort=high no Claude): plano já registrado no doc-043 e revisão do diff antes do Reviewer. Coordenar com a ISSUE-134 e a ISSUE-082.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 No modo ativo, ListenBrainz e importações passam pelo pipeline sem mudar o formato nem a compatibilidade com os clientes; com o interruptor desligado o comportamento é exatamente o da v0.43; a remoção do escritor legado e do interruptor fica para a ISSUE-172, depois do piloto
- [ ] #2 Cada decisão grava o motivo (reason_code, evidência e rule_version) em ingest_events, com a coluna mode (shadow ou active), e a intenção na outbox na mesma transação; colisão com a tripla legada tratada sob BEGIN IMMEDIATE sem ON CONFLICT silencioso nem erro 500
- [ ] #3 Testes de concorrência e de repetições legítimas, e comparação com o resultado do modo sombra da ISSUE-171 sem contagem duplicada na transição; troca da flag com importação em andamento; recuperação após crash entre recibo e resolução; exclusão restrita aos recibos de sombra; EXPLAIN da fila nova
- [ ] #4 Modo Revisar por perfil desligado por padrão, com fallback legado explicável (evidência review_deferred) sem pendência invisível; ativação documentada junto da ISSUE-083
- [ ] #5 Parecer do Advisor (effort high) sobre o diff da ativação registrado; o parecer do plano está no comentário da tarefa e em [PLANO ISSUE-081B] no ai-memory
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: codex
created: 2026-11-16T17:06:00Z

---

Alinhamento após aceite da ISSUE-132: doc-039 e decision-008 exigem preservar recebimentos ambíguos/conflitantes e confirmar ListenBrainz após persistência durável, com revisão separada da deduplicação. O AC1 sobre comportamento observável deve preservar formato e compatibilidade dos clientes, incluindo esta mudança de semântica aprovada para conflitos; não exigir manter o HTTP 500 atual. Playing_now permanece efêmero.

---

author: advisor
created: 2026-11-19T16:15:00Z

---

Parecer do Advisor (Opus, effort medium, 2026-11-19), estudo completo em doc-043. Recomendações 1 a 5 registradas a pedido do usuário como a direção adotada, a confirmar na execução: (1) dividir em 081a (modo sombra: grava recibo e motivos ao lado do caminho atual, sem mudar nenhuma contagem) e 081b (ativação, depois da 133/134), para medir a taxa real de Revisar no banco verdadeiro antes de mudar números; (2) mesma faixa duas vezes no mesmo segundo pela API, sem ID externo: mandar para revisão (contrato §5), não contar como duas; (3) modo Revisar atrás de flag por perfil, desligada até a 083 existir; (4) escuta sem listened_at: aceitar com a hora de chegada marcada como estimated; (5) bug do mbid_mapping (lbz_submit_listen.go:203) corrigido em tarefa própria, fora da 081 (ISSUE-170). Ordem: 081a antes da 133; 133 → 081b e 134 → 082/083; 085/086 por último; migração da 133 será v7 ou posterior. A confirmar na 081a: se a v4 aceita duplicate_decision same sem event_id (inferência do Advisor). Advisor effort=high continua obrigatório sobre o diff do passo 6 (ativação).

---

author: claude
created: 2026-11-19T16:18:00Z

---

Divisão aprovada pelo usuário em 2026-11-19. Parte 1 = ISSUE-171 (modo sombra, antes da 133); parte 2 = esta tarefa (ativação, depois da 171 e da 133). Confirmação do usuário, conferida na migração 004 (linhas 56 e 62): a v4 aceita duplicate_decision same, different e review com event_id NULL, o que viabiliza a parte 1 sem migração. O AC4 original (parecer do Advisor registrado) foi cumprido para o plano pelo doc-043 e permanece para o diff da ativação.

---

author: codex
created: 2026-11-19T16:58:00Z

---

Entradas obrigatórias da ativação após a revisão da ISSUE-171. (3) Excluir recibos de sombra da fila do worker antes de ligá-lo: filtrar os rule_version shadow-v1/shadow-v2, adapter_version ou estado equivalente e verificar que o histórico existente não gera novos eventos. (6) Impor limites por campo, quantidade e tamanho total do snapshot, com política explícita para excedentes e cobertura em teste. (7) Registrar classe/código do erro de recebimento sem incluir payload nem segredos. Medir algumas semanas de modo sombra em produção com a consulta versionada registrada em VERIFICACAO.md da ISSUE-171 antes de habilitar a parte 2.

---

author: codex
created: 2026-11-19T17:17:00Z

---

Confirmação de entradas obrigatórias da ativação. (3) Antes de ativar o worker, excluir da seleção de idx_ingest_queue os recibos de sombra received com closed_at NULL por rule_version, adapter_version ou estado equivalente. (6) Limitar campos, quantidade e tamanho total do snapshot. (7) Registrar classe ou código do erro do recebimento sem expor dados do payload. Incluir também o limite de exclusão da parte 1: quando o segundo UTC de um recibo estimated diverge de listens, a exclusão individual não corresponde ao recibo, que permanece até a limpeza do perfil. A parte 2 deve tratar essa correspondência no desenho da exclusão e testá-la antes da ativação.

---

author: codex
created: 2026-11-19T17:45:00Z

---

Entrada obrigatória para ativação, decisão do usuário de 2026-11-19 na ISSUE-133: com prévia e contagens por perfil, purgar os recibos de sombra da 171 cujo event_id seja nulo antes de ativar a 081. Preservar os recibos vinculados e a configuração das fontes. A 133 preserva os recibos sem vínculo byte a byte e mantém a limpeza legada; a purga de ativação libera chaves de sombra que poderiam bloquear reimportação, inclusive de recibos estimated em outro segundo. Coordenar com leitores e migração de ativação da 135 no mesmo binário.

---

author: codex
created: 2026-11-19T19:06:00Z

---

Entrada obrigatória adicional da ISSUE-133: após a purga aprovada dos recibos de sombra sem evento e no corte coordenado com a 135, a migração de ativação remove trg_ingest_event_inactive (INSERT), trg_ingest_event_inactive_update, trg_ingest_revision_inactive e trg_outbox_inactive.

---

author: ebrgr
created: 2026-11-19T19:21:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-081` na branch `feat/issue-081-pipeline-de-ingestao-com-deduplicacao` criada a partir de `main`.

---

author: codex
created: 2026-11-19T19:26:00Z

---

Fase 1 registrada em backlog/evidencias/issue-081/PLANO.md. Proposta: código dormente na v8, interruptor mestre desligado e opt-in por perfil, sombra em produção antes do corte, purga com prévia e migração coordenada com a ISSUE-135. O plano detalha AC1 por modo, Revisar, compatibilidade, reenvio, transações, relay e ensaio exclusivamente sintético. Nenhum AC marcado. Pausa obrigatória até parecer do Advisor effort high sobre o plano e aprovação explícita do usuário para Fase 2.

---

author: advisor
created: 2026-11-19T19:31:00Z

---

Parecer do Advisor (Opus, effort high, Modo A+B, leitura estática na v0.43 4f9232e) sobre o PLANO.md da Fase 1: REPROVADO COM RESSALVAS. A direção está certa: interruptor desligado, adesão por perfil, sem dual write, sem vínculo heurístico e 200 só após o commit. Corrigir o plano antes da Fase 2.
Falhas críticas: (1) "dormente na v8" (PLANO §5 l.73) não é ativável: os triggers 008:279-297, 221 e 254 abortam accepted/duplicate, event_id, revisões e outbox. Sem a 009, o modo ativo só roda em fixture e a 081 não fecha antes da 135. (2) O índice único completo da tripla (008:79) continua até a 135, mas o plano não diz o que o escritor ativo faz numa colisão de tripla. Ele precisa conferir a tripla sob BEGIN IMMEDIATE e mandar o caso para Revisar ou para o fallback, sem ON CONFLICT silencioso nem 500. (3) deleteListenReceipts (ingest_delete.go:37-63) apaga qualquer recibo no segundo por alias, inclusive recibos ativos pendentes ou vinculados. Restringir o critério à sombra; os vinculados saem pela cascata do event_id. (4) A fila não identifica a sombra positivamente: idx_ingest_queue (008:153) também cobre os pending_duplicate da sombra. (5) A outbox exige destination_id em integrations (008:244), e o relay mora no perfil (profile_store.go:111). (6) "Congelar" o trabalho em voo ao desligar (§2 l.49) deixaria escutas confirmadas com 200 sem contar.
Correções: a 009 ganha ingest_events.mode TEXT NOT NULL DEFAULT 'shadow' CHECK(mode IN('shadow','active')), porque todas as linhas atuais são sombra. Criar um índice parcial da fila WHERE mode='active' AND closed_at IS NULL. Substituir os 4 triggers por guardas por linha (mode='active'), sem ler a flag mutável. Remover a "trilha de compatibilidade" (§2 l.48): no modo desligado, o comportamento fica idêntico à v0.43. O modo vale por lote da API e por job de importação inteiro, lido na transação de admissão. A flag controla só a admissão, e o resolvedor sempre drena os recibos ativos. A purga com prévia vira uma operação root por perfil no momento da ativação, fora do goose, com digest do conjunto conferido sob lock e exportação agregada da medição antes de purgar. Recibo de sombra com escuta equivalente: purgar, sem "Igual" retroativo. Relay até a 082: corpo bruto, como hoje, após o ACK. Gravar a intenção fechada (discarded com motivo pre_082_best_effort), para a 082 não drenar e não haver duplo envio. Mapear BUSY para 503 com Retry-After (ingest_receipts.go:33-35 não mapeia). Limites alinhados ao ListenBrainz: 10 KiB por escuta, 1000 escutas.
Decisões do usuário (recomendação): (a) config mestre (restart) E flag por perfil no banco E schema>=9; reescrever o AC1 ("no modo ativo...") e abrir uma issue para remover o escritor legado. (b) Publicar a 009 com a 081b, mantendo a tripla e a ponte da 133, o que desacopla da 135; a alternativa é manter a 081 aberta até a 135. (c) Revisar desligado = fallback legado explicável: tripla existente vira duplicate do evento, senão o evento é criado, com evidência review_deferred, sem pendência invisível. (d) Origin 'legacy' até a 134. (e) Sem chave, reenvio segue a regra da tripla.
Sequência: C1 costura sem mudança de comportamento → C2 migração 009 [parada: Advisor] → C3 resolvedor transacional → C4 ListenBrainz/importadores/exclusões → C5 operação root de purga e ativação [parada: Advisor high sobre o diff, depois Reviewer] → C6 ensaio sintético e CHANGELOG. Detalhes em ai-memory [PLANO ISSUE-081B].

---

author: claude
created: 2026-11-19T19:58:00Z

---

Decisões do usuário em 2026-11-19, após o parecer do Advisor sobre o plano da Fase 1 (REPROVADO COM RESSALVAS, 6 falhas a corrigir no plano): o usuário ACEITOU TODAS as correções e recomendações (a) a (f), inclusive (b). (a) Interruptor mestre (config lida no restart) mais flag por perfil; o AC1 passa a valer 'no modo ativo' (reescrito); abrir tarefa para remover o escritor legado e o interruptor depois do piloto (ISSUE-172). (b) A migração 009 sai junto com a 081b, mantendo a tripla e a ponte da 133, o que desacopla a 081 da 135 e muda a ordem aprovada na 133; a segurança passa a depender da coluna mode por linha e da tripla como rede contra contagem dupla. (c) Revisar desligado: fallback legado explicável (se a tripla já existe, o recibo vira duplicate do evento existente; senão cria o evento; ambos com evidência review_deferred para a 083). (d) Origem permanece legacy até a 134 cadastrar as novas em event_origins. (e) Reenvio sem chave segue a regra da tripla: mesma faixa no mesmo segundo não aumenta a contagem em relação ao legado. (f) Algumas semanas de modo sombra em produção antes de ativar o primeiro perfil. Correções ao plano: 009 com ingest_events.mode, índice parcial de fila WHERE mode=active AND closed_at IS NULL, guardas por mode da própria linha, tabela de flags por perfil, tripla e ponte mantidas, bootstrap/backup/ingest_schema atualizados; modo efetivo lido na transação de admissão (lote da API ou job de importação inteiro); resolvedor (inline e laço de recuperação com lease e CAS) sempre termina recibos ativos, de modo que desligar volta ao legado sem perda; remover a trilha de compatibilidade (terceiro escritor); purga com prévia como operação do root por perfil, fora do goose, com digest conferido sob lock e exportação agregada da medição, sem Igual retroativo; relay legado (corpo bruto após o ACK) até a ISSUE-082, com a intenção na outbox já fechada (discarded, pre_082_best_effort) e integração de relay por perfil; validar o lote inteiro e responder 400 sem gravar, 503 com Retry-After para BUSY e armazenamento, 500 para erro interno inesperado, limites do ListenBrainz (10 KiB por escuta, 1000 escutas); exclusão restrita à sombra (ingest_delete.go:37-63); testes de colisão de tripla, troca da flag, crash entre recibo e resolução e EXPLAIN. Sequência da Fase 2: C1 separar resolução e gravação (sem mudar comportamento), C2 migração 009 [parada: Advisor], C3 resolvedor transacional, C4 ListenBrainz, importadores e exclusões, C5 purga e ativação do root [parada: Advisor high sobre o diff, depois Reviewer], C6 ensaio sintético, evidências e CHANGELOG. STAND BY por decisão do usuário: a Fase 2 só será liberada depois que o limite de uso do Codex renovar.

---

author: codex
created: 2026-11-20T02:14:00Z

---

Fase 2, parada após C2 para parecer do Advisor. C1 (c53408f) separa a associação de catálogo da escrita legada e mantém o lock e o comportamento dos chamadores. C2 inclui a migração 009, modo imutável por recibo, fila parcial ativa, flags por perfil com padrão 0, guardas por linha e backup v8. O teste de upgrade confirma a preservação dos dados, o índice usado em EXPLAIN e a resolução de um recibo ativo com admissão desligada. go test passou em SQLite, dedup, catálogo, importadores, engine, handlers, middleware e administração com TUNETRAIL_WEB_DIR apontando para os assets da worktree. Nenhum AC marcado nesta parada.

---

author: codex
created: 2026-11-20T14:40:00Z

---

Fase 2: usuário registrou APROVADO para C1+C2, sem item obrigatório nesses commits. Endurecimento das guardas no fbab8a6. C3 (4660905) implementa resolução por lease e CAS sob BEGIN IMMEDIATE, colisão da tripla e decisão/revisão/outbox atômicas. C4 (8512a94) integra ListenBrainz, os importadores e exclusões. compareNearby filtra mode=shadow para preservar a série de comparação do modo sombra; a resolução ativa consulta eventos vivos. Replay entre modos conserva o dono de cada chave. recording_identity_test.go conserva o corte histórico em v8 com UpTo 8; a ISSUE-135 deve partir de v9. A outbox possui guarda de INSERT; a ISSUE-082 deve considerar guarda de UPDATE de event_id/profile_id. C5 expõe GET e PUT root em /apis/web/v1/profiles/{slug}/ingest-activation: exportar a prévia agregada, confirmar com o digest e active=true, review=false; a purga da sombra sem vínculo e a ativação são atômicas. Antes do piloto, manter a medição de produção aprovada e coordenar a janela de corte. A reativação por perfil após o modo sombra exige nova prévia. O mestre TUNETRAIL_INGEST_ACTIVE é lido no restart. go test passou nos pacotes afetados e o Graphify AST foi atualizado. Parada para Advisor high e depois Reviewer. C6, make check, medição de produção e marcação dos ACs ficam pendentes.

---

author: codex
created: 2026-11-20T17:33:00Z

---

Correção de autoria do comentário anterior: o Advisor aprovou C1+C2, sem itens obrigatórios nesses commits. O usuário comunicou o parecer e autorizou a continuação.

---

author: codex
created: 2026-11-20T17:40:00Z

---

Ressalvas do Advisor high sobre C3–C5 tratadas em commits separados: d1487bd admite os recibos do Koito; b02dffa persiste metadados de associação e compara retries v1 com a projeção v1; 9953dee rejeita identidade incompleta e refaz a associação após remoção da faixa fixada; 3208627 rejeita review=true até existir fila de revisão; f7c9590 fecha recibos ativos received/retryable do perfil no mesmo commit da exclusão de histórico e invalida o lease anterior; 446bde4 devolve 403 quando a sessão root não é válida; 017f0ae exibe Pendentes e Duplicadas na importação ativa; 06253e1 cobre Koito, ListenBrainz com mbid_mapping, faixa apagada, lease vencido com disputa entre conexões, intenção relay:<id>, 400 sem escrita do prefixo e 503 com Retry-After em BUSY ou falha de COMMIT. A inclusão manual permanece no escritor legado e está fora do escopo da ISSUE-081b. R1–R5 e R7 estão documentadas na evidência vinculada, incluindo o ensaio de 300 mil recibos e a janela de manutenção. go test passou nos pacotes SQLite/DB, ingest, dedup, importadores, catálogo, admin, adminweb, adminui, i18n e engine com TUNETRAIL_WEB_DIR apontando para os assets da worktree. Graphify AST atualizado. Parada para o Reviewer. C6, CHANGELOG e make check ficam pendentes.

---

author: codex
created: 2026-11-20T17:57:00Z

---

Correção do obrigatório do Reviewer em 95b4a83: ApplyProfileDelete fecha recibos ativos abertos somente quando a faixa fixada pertence à seleção excluída ou, sem faixa fixada, quando artista e título correspondem aos aliases da mesma faixa selecionada. O teste prova que o recibo relacionado perde o lease e que o recibo sem relação permanece aberto e resolve em seguida. R-c em 80f7ef0 mapeia ErrUnavailable para 503 com Retry-After no painel, com falha de COMMIT injetada. R-d em f69946b exclui CatalogTrackID do digest v1. R-e em 0be535a verifica a validação prévia já existente do lote Koito, incluindo ausência de efeitos no catálogo quando o segundo item é inválido. R-a e R-b estão documentadas para medir e decidir na C6. go test passou nos pacotes afetados com os assets da worktree. Graphify AST atualizado. Parada para releitura do diff. C6, CHANGELOG e make check continuam pendentes.

---
<!-- COMMENTS:END -->
