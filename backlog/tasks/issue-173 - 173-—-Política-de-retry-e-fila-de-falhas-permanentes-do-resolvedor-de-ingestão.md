---
id: ISSUE-173
title: 173 — Política de retry e fila de falhas permanentes do resolvedor de ingestão
status: Aberta
assignee: []
created_date: '2026-11-20T18:20:00Z'
updated_date: '2026-11-20T18:23:00Z'
labels:
  - ingestao
milestone: m-11
dependencies:
  - ISSUE-081
references:
  - backlog/evidencias/issue-081/ENSAIO-C6.md
type: feature
ordinal: 186000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: ressalva R-a do Reviewer e medição da C6 da ISSUE-081 (backlog/evidencias/issue-081/ENSAIO-C6.md, seção R-a). No modo ativo o resolvedor incrementa attempt_count mas nenhum código usa o contador: um recibo confirmado com 200 cuja resolução falha de forma persistente volta a retryable a cada minuto, para sempre e sem aparecer para o operador (medido: tentativas em 0,99 s, 60,99 s e 121,00 s; recuperou na 4ª tentativa quando a falha foi removida). Proposta medida para aprovar: até 5 tentativas automáticas para falhas transitórias, com espera de 1, 2, 4 e 8 minutos; na 5ª falha, reter recibo, chave, motivo e contagem numa fila durável de falhas permanentes, com alerta e reprocessamento explícito após a correção; encerrar com motivo registrado os erros de entrada reconhecidamente determinísticos; nunca descartar recibo confirmado (recuperável sempre). Decisões do usuário em aberto: teto e esperas, onde mostrar a fila (menu de saúde/admin), quem reprocessa (root) e se a fila de revisão (ISSUE-083) compartilha a mesma tela. Inclui R-b (decisão do usuário em 20/11/2026): erro genérico antes de o modo ser conhecido deixa de dar 500 e segue pelo caminho legado. Pré-requisito: ISSUE-081 publicada. Deve ser concluída antes de ativar o piloto da ingestão. Escopo a refinar com o Advisor (effort high: concorrência, migração) antes de implementar.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 O resolvedor limita as tentativas automáticas (teto e espera crescente aprovados) e move o recibo esgotado para uma fila durável de falhas permanentes sem perder chave, motivo ou contagem
- [ ] #2 O operador root enxerga a fila e reprocessa um recibo de forma explícita; o reprocessamento reaproveita a mesma resolução transacional
- [ ] #3 Erros de entrada determinísticos são encerrados com motivo registrado; falhas transitórias continuam recuperáveis
- [ ] #4 Ensaio sintético com falha persistente e transitória, crash e retomada por lease, sem escuta confirmada perdida; Advisor (effort high) registrado
- [ ] #5 R-b: com o interruptor mestre ligado, um erro genérico (que não é de armazenamento) antes de o modo do perfil ser conhecido segue pelo caminho legado em vez de responder 500; erros de armazenamento e BUSY continuam em 503 com Retry-After; testado por classe de erro
<!-- AC:END -->
