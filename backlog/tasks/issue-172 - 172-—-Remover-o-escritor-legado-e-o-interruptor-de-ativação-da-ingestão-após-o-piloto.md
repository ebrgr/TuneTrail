---
id: ISSUE-172
title: >-
  172 — Remover o escritor legado e o interruptor de ativação da ingestão após o
  piloto
status: Aberta
assignee: []
created_date: '2026-11-19T19:58:00Z'
updated_date: '2026-11-21T23:22:00Z'
labels:
  - ingestao
milestone: m-11
dependencies:
  - ISSUE-081
  - ISSUE-135
priority: low
type: chore
ordinal: 185000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: decisão do usuário em 2026-11-19 sobre a ISSUE-081 (parte 2), item (a) do parecer do Advisor. A 081b entrega o pipeline atrás de um interruptor (mestre mais flag por perfil), desligado por padrão, mantendo o escritor legado como fonte da verdade até a medição de algumas semanas de modo sombra em produção e o piloto. Depois de um piloto bem-sucedido e do corte coordenado com a ISSUE-135 (leitores por ocorrência), esta tarefa remove o escritor legado de listens, o interruptor e as flags por perfil, e a tripla legada idx_listens_legacy_identity se a 135 já tiver substituído os leitores. Escopo a refinar com o Advisor (effort high, concorrência e migração de dados): migração que remove o que a 081b deixou como compatibilidade, reversão só por backup, ensaio sintético e, se houver, validação em cópia de snapshot autorizado. Pré-requisitos: ISSUE-081 e ISSUE-135 concluídas e piloto avaliado.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 O escritor legado de listens e o interruptor de ativação são removidos sem perda de escutas, com ensaio sintético e o Advisor (effort high) registrado
- [ ] #2 Contagens por perfil e digests iguais antes e depois, reversão por backup comprovada
- [ ] #3 Os limiares aprovados (p95 de ACK < 200 ms, visibilidade < 10 s, < 0,1% de 503, até 15 escutas/s) são medidos em ensaio sintético com carga, no lugar das 3 semanas de sombra, e registrados
- [ ] #4 Backup verificado e restauração comprovada antes do corte; ISSUE-181 e ISSUE-182 concluídas antes de ligar a ingestão ativa
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-21T23:22:00Z

---

Decisão do usuário (21/11/2026): mudança de plano. Depois de concluir a ISSUE-181 (e a 182), ligar o código novo e remover o legado sem esperar as 3 semanas de sombra. Consequências registradas: (1) os limiares aprovados passam a ser comprovados por ensaio sintético com carga e por cópia de snapshot autorizado, não por produção; (2) o corte só é reversível por backup, então o backup verificado e a restauração testada são pré-requisito; (3) Advisor (effort high, concorrência e migração de dados) obrigatório antes do aceite; (4) a janela de manutenção de 10 minutos continua. Risco aceito pelo usuário: sem o período de sombra, problemas de dados reais só aparecem depois do corte.

---
<!-- COMMENTS:END -->
