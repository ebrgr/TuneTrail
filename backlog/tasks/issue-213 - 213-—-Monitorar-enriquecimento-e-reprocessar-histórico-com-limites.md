---
id: ISSUE-213
title: 213 — Monitorar enriquecimento e reprocessar histórico com limites
status: Aberta
assignee: []
created_date: '2026-11-24T11:43:00Z'
updated_date: '2026-11-28T02:20:00Z'
labels:
  - operacao
  - ingestao
  - ui
milestone: m-13
dependencies:
  - ISSUE-055
  - ISSUE-214
references:
  - backlog/evidencias/issue-055/ESTUDO-COMPLEMENTAR.md
type: feature
ordinal: 226000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Permitir enriquecer gradualmente o histórico existente com limites configuráveis de requisições por provedor e apresentar o progresso e a situação do enriquecimento na central de monitoramento ligada à central de saúde. Coordenar com o subsistema das ISSUE-174, 175 e 055, sem bloquear a ingestão e preservando o isolamento por perfil.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Processar histórico de forma gradual e retomável, sem gerar trabalho duplicado, respeitando limites configuráveis por provedor.
- [ ] #2 Mostrar progresso, provedor, resultados, falhas e estado de espera na central de monitoramento, com acesso pela central de saúde.
- [ ] #3 Respeitar permissões e isolamento de perfis; indisponibilidade de provedor não impede registrar escutas.
- [ ] #4 O orçamento compartilhado do MusicBrainz ganha classe de prioridade, de modo que o enriquecimento em lote do histórico fique abaixo do trabalho de escutas novas e do resolvedor, e um teto por minuto configurável por provedor (requisito que saiu da ISSUE-216 por decisão do usuário em 26/11/2026; hoje o orçamento é uma fila FIFO com um limite global em queue/queue.go)
- [ ] #5 Listar na central de saúde, por perfil, as faixas, lançamentos e artistas sem dado em todas as fontes disponíveis (marcas da ISSUE-217) e permitir reconsultá-los, inclusive com um serviço habilitado depois
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-26T20:20:00Z

---

26/11/2026: acrescentado o critério de prioridade e teto por minuto no orçamento compartilhado, que saiu da ISSUE-216. Não há importação de arquivo na fila de enriquecimento hoje (só escutas novas pela API); este é o ponto em que o trabalho em lote do histórico passa a competir com as escutas novas pela mesma taxa de 1 requisição por segundo do MusicBrainz.

---

author: claude
created: 2026-11-27T14:37:00Z

---

27/11/2026, do parecer do Advisor da ISSUE-216: com a fila compartilhada do MusicBrainz cheia por uma rajada do resolvedor (a fila tem limite), a espera de um Lookup passa do timeout de 20 s e a etapa devolve Failed; na cadeia da ISSUE-214 um job com Failed fecha como terminal. O enriquecimento em lote do histórico precisa tratar esse caso: reprocessar jobs 'failed' e dar prioridade ao trabalho de escutas novas. Deve entrar nos critérios de prioridade e de reprocessamento desta issue.

---

author: claude
created: 2026-11-28T02:20:00Z

---

Decisão do usuário em 27/11/2026: o que ficar sem dado deve ser acessível na Central de saúde para reconsulta posterior. A ISSUE-217 cria o registro (migração 027); a listagem e a reconsulta são desta tarefa.

---
<!-- COMMENTS:END -->
