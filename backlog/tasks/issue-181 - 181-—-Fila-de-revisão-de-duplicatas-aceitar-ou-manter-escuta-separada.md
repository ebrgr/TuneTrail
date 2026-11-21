---
id: ISSUE-181
title: '181 — Fila de revisão de duplicatas: aceitar ou manter escuta separada'
status: Aberta
assignee: []
created_date: '2026-11-21T16:07:00Z'
updated_date: '2026-11-21T16:17:00Z'
labels:
  - ingestao
milestone: m-11
dependencies:
  - ISSUE-083
references:
  - backlog/evidencias/ISSUE-083/PARECER-REVIEWER.md
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
- [ ] #1 Verificar se o backend já produz casos em revisão e se a flag de Revisar por perfil pode ser ativada
- [ ] #2 Pendências de revisão listadas na central de saúde, por perfil, sem expor segredos
- [ ] #3 Aceitar como duplicata e Manter como escuta separada gravam a decisão em uma única transação, com CSRF
- [ ] #4 Testes de isolamento entre perfis e de decisão repetida ou concorrente
- [ ] #5 Visual consistente com a central de saúde, responsivo e com hover
- [ ] #6 Medir o custo das consultas da central de saúde (fontes e decisões) com volume realista de eventos e, se necessário, limitar a janela de tempo ou indexar; risco 1 do parecer da ISSUE-083
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-21T16:17:00Z

---

Herdado do parecer do Reviewer da ISSUE-083 (21/11/2026): o risco 1 (healthSources e healthDecisions percorrem todos os eventos do perfil a cada abertura da página, sem medição) passou para esta tarefa, pois a fila de revisão vai ler as mesmas tabelas. Os riscos 2 e 3 do parecer foram resolvidos na 083 (item no menu; AC do acesso reescrito). Parecer atualizado em backlog/evidencias/ISSUE-083/PARECER-REVIEWER.md.

---
<!-- COMMENTS:END -->
