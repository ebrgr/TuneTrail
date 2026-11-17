---
id: ISSUE-165
title: 165 — Limitar o cache de imagens pelo tamanho total e apagar as mais antigas
status: Aberta
assignee: []
created_date: '2026-11-17T17:34:00Z'
updated_date: '2026-11-17T17:35:00Z'
labels:
  - imagens
  - operacao
  - config
milestone: m-10
dependencies: []
documentation:
  - doc-042
type: feature
ordinal: 178000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: pedido do usuário em 2026-11-17; estudo em doc-042. Hoje o cache de imagens (image_cache/) só tem limites POR IMAGEM (download de 10 MiB e 10 s, 10000 px por lado e 40 MP, original normalizado para 1200 px em WebP) e nenhum limite do tamanho total, nem por idade: só a coleta de órfãs, uma vez por partida, apaga imagens sem associação no banco. Os derivados (64, 128, 300, 640 e 1000 px) são gerados sob demanda e nunca saem. A opção KOITO_ENABLE_FULL_IMAGE_CACHE é morta (nunca lida). Medição local: cerca de 330 KiB por imagem com todos os tamanhos (1.000 imagens ≈ 330 MiB; 10.000 ≈ 3,3 GiB). Proposta: um teto de tamanho total configurável; ao se aproximar do teto, apagar primeiro os derivados mais antigos (regeneráveis) e depois as pastas de originais remotos mais antigas (rebaixa sob demanda a partir de image_source), com marca d'água e histerese (acima de 100%, apagar até 90%). Nunca apagar avatares nem uploads (sem URL de origem), nem imagens com reserva de publicação ativa. Último acesso gravado pelo aplicativo, com intervalo mínimo, em vez de atime. Perguntas abertas ao usuário no doc-042 §7 (teto padrão, critério de antiguidade, apagar originais, onde configurar). Coordenar com ISSUE-117 (backfill), 153 e 127 (configuração), 112 (logs) e 102/103 (volumes e backup).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Decisões do usuário sobre as 4 perguntas do doc-042 §7 registradas e parecer do Advisor
- [ ] #2 Teto de tamanho total configurável (0 = sem limite) e piso mínimo; ao passar do teto, apaga derivados e depois originais remotos mais antigos até a marca d'água
- [ ] #3 Avatares, uploads e imagens com reserva de publicação nunca são apagados (testes)
- [ ] #4 Original remoto apagado é baixado de novo sob demanda sem laço de remoção, e a imagem servida na requisição nunca é removida
- [ ] #5 Coleta concorrente com gravação e com a coleta de órfãs coberta por teste; uso do cache e remoções registrados em log
- [ ] #6 Opção morta KOITO_ENABLE_FULL_IMAGE_CACHE removida ou documentada como obsoleta; docs e CHANGELOG atualizados
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-17T17:35:00Z

---

Estudo feito (doc-042): não existe limite de tamanho total hoje; a proposta aguarda as 4 decisões do usuário.

---
<!-- COMMENTS:END -->
