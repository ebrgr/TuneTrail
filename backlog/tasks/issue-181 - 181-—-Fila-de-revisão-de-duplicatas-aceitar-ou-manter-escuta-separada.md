---
id: ISSUE-181
title: '181 — Fila de revisão de duplicatas: aceitar ou manter escuta separada'
status: Aguardando aceite
assignee:
  - '@ebrgr'
created_date: '2026-11-21T16:07:00Z'
updated_date: '2026-11-22T01:03:00Z'
labels:
  - ingestao
milestone: m-11
dependencies:
  - ISSUE-083
references:
  - backlog/evidencias/ISSUE-083/PARECER-REVIEWER.md
  - backlog/evidencias/issue-181/VERIFICACAO.md
  - backlog/evidencias/issue-181/CORRECOES.md
priority: medium
type: feature
ordinal: 194000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: pendência registrada na ISSUE-083 (central de saúde), comentário #3 (proposta de 2026-11-19) e doc-043 (081b). A central de saúde da 083 é somente leitura: mostra fontes, fila do relay e decisões já tomadas, mas não permite decidir nada.

Situação do backend (lida no código em 2026-11-21): o pipeline de ocorrência já grava recibos pendentes de revisão (state pending_duplicate, identity_status pending, duplicate_decision review, internal/db/sqlite/ingest_occurrence.go writeOccurrenceDecision), com evidência de candidatos. Não existe rota, ação nem tela que resolva esses casos, e não foi verificado se o modo Revisar por perfil (flag desligada por padrão, previsto na 081b) já pode ser ativado.

Escopo proposto: seção Pendências de revisão na central de saúde (/admin/health), com Aceitar como duplicata, Manter como escuta separada e Ver evidência; resolução gravada em uma única transação (decisão, evento e intenção na outbox, sem dual write); flag de Revisar por perfil, desligada por padrão; acesso do dono ao próprio perfil e do root a todos, com CSRF e sem expor tokens nem corpo bruto; visual igual ao da central de saúde e do diálogo de edição de mídia. Fora do escopo: indicadores da ISSUE-097 e integração com os logs da ISSUE-112.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Verificar se o backend já produz casos em revisão e se a flag de Revisar por perfil pode ser ativada
- [x] #2 Pendências de revisão listadas na central de saúde, por perfil, sem expor segredos
- [x] #3 Aceitar como duplicata e Manter como escuta separada gravam a decisão em uma única transação, com CSRF
- [x] #4 Testes de isolamento entre perfis e de decisão repetida ou concorrente
- [x] #5 Visual consistente com a central de saúde, responsivo e com hover
- [x] #6 Medir o custo das consultas da central de saúde (fontes e decisões) com volume realista de eventos e, se necessário, limitar a janela de tempo ou indexar; risco 1 do parecer da ISSUE-083
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-21T16:17:00Z

---

Herdado do parecer do Reviewer da ISSUE-083 (21/11/2026): o risco 1 (healthSources e healthDecisions percorrem todos os eventos do perfil a cada abertura da página, sem medição) passou para esta tarefa, pois a fila de revisão vai ler as mesmas tabelas. Os riscos 2 e 3 do parecer foram resolvidos na 083 (item no menu; AC do acesso reescrito). Parecer atualizado em backlog/evidencias/ISSUE-083/PARECER-REVIEWER.md.

---

author: ebrgr
created: 2026-11-21T23:13:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-181` na branch `feat/issue-181-fila-de-revisao-de-duplicatas` criada a partir de `main`.

---

author: codex
created: 2026-11-21T23:27:00Z

---

Verificação: ambos os resolvedores já geravam pending_duplicate. A flag legada era bloqueada na API e o resolvedor de ocorrência não a consultava. A central passou a oferecer controle por perfil, fila paginada e decisão transacional com CSRF e outbox elegível. Testes de isolamento, concorrência, rollback e reenvio passaram. Referência ISSUE-075 regenerada e validada no Orca. EXPLAIN com 300 mil recibos confirmou a janela de fontes e os índices do schema v16. Conflitos sem identidade ou candidato permanecem visíveis com evidência e sem ação de decisão.

---

author: claude
created: 2026-11-21T23:32:00Z

---

Reviewer: sem BUG; 4 RISCOs (padrão da flag Revisar com a flag desligada aceita como separada; janela de 10 mil recibos em fontes; autor da decisão sem ID; migração recria índice legado). Testes sqlite -race, engine, adminui e webui OK. Ver backlog/evidencias/issue-181/PARECER-REVIEWER.md.

---

author: advisor
created: 2026-11-21T23:35:00Z

---

Advisor (effort high): REPROVADO COM RESSALVAS. Falha crítica: com o Revisar desligado (padrão), uma reentrega no mesmo segundo é aceita como nova escuta (finishOccurrence ignora os candidatos), contrariando o doc-039 e o resolvedor legado. Melhorias: ação em massa na fila, resumo por fonte no lugar da janela de 10 mil recibos (ISSUE-184), autor da decisão. Ver backlog/evidencias/issue-181/PARECER-ADVISOR.md.

---

author: codex
created: 2026-11-22T00:26:00Z

---

Correções verificadas: colisão exata com Revisar desligado volta a registrar duplicate/same, e a janela de ±2 s aceita escutas separadas. A central tem contador e ações em massa de até 25 recibos, com CAS, CSRF e ID da conta na revisão. O resumo paginado do schema v17 preserva fontes antigas. Testes focados, corrida, EXPLAIN com 300 mil recibos, referência ISSUE-075 e make check passaram.

---

author: claude
created: 2026-11-22T00:30:00Z

---

Reviewer (reauditoria do commit b0c1b66): sem BUG executado; RISCO ALTO não executado: o braço de colisão exata no mesmo segundo em occurrenceListenCandidates não exclui repetições legítimas (IDs externos confiáveis distintos; linhas idênticas de arquivo com ordinais distintos), que com o Revisar desligado viram duplicate/same e deixam de ser contadas. Demais correções OK. Ver PARECER-REVIEWER.md.

---

author: codex
created: 2026-11-22T00:53:00Z

---

Verificação da correção final: o filtro de evidência de eventos distintos agora vale também para o segundo exato. Testes cobrem reentrega sem ID, ordinais distintos no mesmo arquivo, IDs confiáveis distintos, reenvio da mesma chave, isolamento entre perfis e Revisar ligado e desligado. Para uma reentrega por ID confiável fora de arquivo, depois de resolvido o recibo original, Revisar abre uma única pendência que preserva a chave original.

---

author: claude
created: 2026-11-22T00:58:00Z

---

Reviewer (reauditoria final, commit 247a0f8): sem BUG; o RISCO ALTO anterior foi resolvido (repetições legítimas preservadas no mesmo segundo; testes a-f). Testes sqlite -race, engine, adminui e webui OK. Decisão de produto pendente: reenvio do mesmo ID externo confiável com Revisar ligado abre uma pendência (uma por original). Ver PARECER-REVIEWER.md.

---

author: ebrgr
created: 2026-11-22T01:03:00Z

---

Verificação: make check 13/13 E2E (Codex); go test -race sqlite/engine/adminui/webui OK; Reviewer sem BUG após 2 reauditorias; Advisor reprovou com ressalvas e os itens foram corrigidos (reauditoria pelo Reviewer, aceita pelo usuário). ACs 1 a 6 marcados. Decisão de produto pendente: reenvio do mesmo ID externo confiável com Revisar ligado.

---
<!-- COMMENTS:END -->
