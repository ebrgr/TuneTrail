---
id: ISSUE-081
title: '081 — Pipeline de ingestão com deduplicação explicável: ativação (parte 2)'
status: Aberta
assignee: []
created_date: '2026-11-14T00:39:00Z'
updated_date: '2026-11-19T19:06:00Z'
labels:
  - ingestao
milestone: m-11
dependencies:
  - ISSUE-171
  - ISSUE-133
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
- [ ] #1 ListenBrainz e importações passam pelo pipeline ativo sem mudar o formato nem a compatibilidade com os clientes; a escrita direta em listens deixa de existir e o 500 em conflito vira resposta definida
- [ ] #2 Cada decisão grava o motivo (reason_code, evidência e rule_version) em ingest_events e a intenção na outbox na mesma transação
- [ ] #3 Testes de concorrência e de repetições legítimas, e comparação com o resultado do modo sombra da ISSUE-171 sem contagem duplicada na transição
- [ ] #4 Modo Revisar atrás de flag por perfil desligada por padrão; ativação documentada junto da ISSUE-083
- [ ] #5 Parecer do Advisor (effort high) sobre o diff da ativação registrado; plano já registrado no doc-043
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
<!-- COMMENTS:END -->
