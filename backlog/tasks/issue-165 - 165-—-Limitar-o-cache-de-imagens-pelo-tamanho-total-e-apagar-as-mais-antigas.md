---
id: ISSUE-165
title: 165 — Limitar o cache de imagens pelo tamanho total e apagar as mais antigas
status: Planejada
assignee: []
created_date: '2026-11-17T17:34:00Z'
updated_date: '2026-11-17T17:51:00Z'
labels:
  - imagens
  - operacao
  - config
milestone: m-10
dependencies:
  - ISSUE-153
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
- [ ] #1 Decisões do usuário registradas (doc-042 §9) e parecer do Advisor sobre a limpeza concorrente
- [ ] #2 Teto de tamanho total no conf.toml, com valor padrão (1024 MiB sugerido), alterável apenas pelo administrador root; 0 = sem limite; piso mínimo
- [ ] #3 Ao passar do teto, apaga os derivados e depois os originais remotos mais antigos por último acesso, até a marca d'água (90%)
- [ ] #4 Avatares, uploads e imagens com reserva de publicação nunca são apagados (testes)
- [ ] #5 Original remoto apagado é baixado de novo sob demanda sem laço de remoção, e a imagem servida na requisição nunca é removida
- [ ] #6 Coleta concorrente com gravação e com a coleta de órfãs coberta por teste; uso do cache e remoções registrados em log
- [ ] #7 Opção morta KOITO_ENABLE_FULL_IMAGE_CACHE removida ou documentada como obsoleta; docs e CHANGELOG atualizados
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-17T17:35:00Z

---

Estudo feito (doc-042): não existe limite de tamanho total hoje; a proposta aguarda as 4 decisões do usuário.

---

author: ebrgr
created: 2026-11-17T17:51:00Z

---

Decisões do usuário (2026-11-17), registradas no doc-042 §9: 1) teto padrão, configurável pelo administrador root; 2) mais antigo por último acesso; 3) aceita a orientação do estudo (derivados primeiro, depois originais remotos com nova baixa sob demanda; nunca avatares e uploads); 4) configuração no conf.toml, reaproveitando o mecanismo da ISSUE-153 (por isso passa a depender dela). Status: Planejada.

---
<!-- COMMENTS:END -->
