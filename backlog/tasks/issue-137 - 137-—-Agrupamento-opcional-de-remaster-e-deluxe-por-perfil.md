---
id: ISSUE-137
title: 137 — Agrupamento opcional de remaster e deluxe por perfil
status: Aberta
assignee: []
created_date: '2026-11-15T16:43:00Z'
updated_date: '2026-11-15T16:44:00Z'
labels:
  - catalogo
  - ui
milestone: m-13
dependencies:
  - ISSUE-135
  - ISSUE-096
documentation:
  - doc-036
type: feature
ordinal: 150000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Recorte 7 do estudo da ISSUE-116 (doc-036), Koito #360. Agrupamento reversível de remaster e deluxe por perfil, desligado por padrão, coerente com os grupos de artistas da ISSUE-096. Decisão do usuário: convive com a fusão global (ISSUE-138), e uma configuração do servidor define quais formas ficam disponíveis.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Prévia e desfazer por perfil; lançamentos e IDs originais preservados
- [ ] #2 Bônus de deluxe e gravações ao vivo não entram automaticamente
- [ ] #3 Totais brutos invariantes; rankings com e sem grupo testados
- [ ] #4 Configuração do servidor liga ou desliga a opção
<!-- AC:END -->
