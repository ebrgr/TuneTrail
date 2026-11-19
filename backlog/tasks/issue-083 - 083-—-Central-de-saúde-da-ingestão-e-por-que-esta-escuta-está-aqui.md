---
id: ISSUE-083
title: 083 — Central de saúde da ingestão e "por que esta escuta está aqui"
status: Aberta
assignee: []
created_date: '2026-11-14T00:39:00Z'
updated_date: '2026-11-19T16:15:00Z'
labels:
  - admin
  - ingestao
milestone: m-11
dependencies:
  - ISSUE-082
priority: medium
type: feature
ordinal: 83000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: doc-010 §4.2. `/admin/health` por perfil (último envio, cliente, fila, falhas e motivos, sem tokens) e, na página da escuta, origem, horário recebido, regra aplicada e estado de entrega.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Central de saúde por perfil sem expor segredos
- [ ] #2 Explicação por escuta acessível ao admin
- [ ] #3 Testes de isolamento entre perfis
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-14T03:48:00Z

---

Coordenar com a ISSUE-112 (logs): a central de saúde mostra a contagem de erros e avisos da última hora com link para /admin/logs filtrado.

---

author: claude
created: 2026-11-15T15:18:00Z

---

A página desta tarefa segue o padrão visual comum da remodelação da administração (ISSUE-074) e entra como item da segunda barra (ISSUE-123).

---

author: claude
created: 2026-11-19T16:15:00Z

---

Proposta registrada a pedido do usuário (2026-11-19), a partir do estudo da ISSUE-081 (doc-043): reunir as checagens e revisões da ingestão num lugar próprio, a central de saúde por perfil (/admin/health ou entrada Saúde no menu de gestão do perfil), em vez de espalhá-las pelas telas. Seções sugeridas: (1) Pendências de revisão: fila de duplicatas prováveis e identidades ambíguas geradas pela 081b, com Aceitar como duplicata, Manter como escuta separada e Ver evidência; (2) Últimas decisões da deduplicação: por que esta escuta está aqui, com reason_code, evidência (Δ de tempo, divergência de álbum, ID externo) e rule_version; (3) Fontes e último envio: integração, cliente, horário recebido, estado de entrega e precisão do horário (second ou estimated); (4) Fila e falhas: outbox/relay da 082, tentativas e motivos, sem expor tokens; (5) Verificações automáticas: escutas com horário estimado, lotes rejeitados, relógio adiantado, perfis sem envio há N dias, e os indicadores de qualidade da ISSUE-097. Pontos de atenção: a 081b só deve ativar o modo Revisar (flag por perfil desligada por padrão) quando esta central existir, senão as pendências ficam invisíveis; o acesso passa a ser do dono para o próprio perfil e do root para todos (alinhado à autorização da 160), em vez de só admin; dados sensíveis (tokens, corpo bruto) nunca na tela; a visibilidade da própria central não depende de require_login (é gestão, sempre exige login, ISSUE-161). Sugestão de ajuste de escopo ao replanejar: acrescentar um critério para a fila de revisão e dependência da 081 (081b), e ajustar o AC2 de admin para dono e root.

---
<!-- COMMENTS:END -->
