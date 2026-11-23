---
id: ISSUE-193
title: 193 — Normalizar metadados na ingestão com regras por perfil e prévia
status: Aberta
assignee: []
created_date: '2026-11-22T20:40:00Z'
updated_date: '2026-11-23T00:38:00Z'
labels:
  - ingestao
  - catalogo
  - admin
dependencies:
  - ISSUE-081
documentation:
  - doc-045
priority: medium
type: feature
ordinal: 206000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Avaliar regras literais de substituição exata e remoção de prefixo ou sufixo por campo, com escopo opcional por origem. Usar recibos de exemplo para medir o efeito na associação ao catálogo e na deduplicação antes de ativar regras para eventos futuros.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 A prévia mostra entrada, saída, regras aplicadas em ordem e efeito na deduplicação sem gravar alterações.
- [ ] #2 Preservar o recibo original, isolar regras por perfil e registrar a versão da regra ou o resultado aplicado para reproduzir retries.
- [ ] #3 Definir e testar o tratamento de campos vazios, regras sobrepostas, identificadores conflitantes, eventos históricos e alterações de regra entre tentativas.
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-23T00:38:00Z

---

Decisão do usuário (22/11/2026): esta issue está no mínimo que altera o banco antes do primeiro upgrade de produção e antecede a consolidação das migrações (ISSUE-195). Migração aditiva agora; depois dobrada no arquivo único da 195.

---
<!-- COMMENTS:END -->
