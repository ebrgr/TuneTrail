---
id: ISSUE-153
title: 153 — Persistir as opções do painel de tema no conf.toml
status: Aberta
assignee: []
created_date: '2026-11-16T18:57:00Z'
updated_date: '2026-11-17T12:47:00Z'
labels:
  - config
  - ui
milestone: m-10
dependencies:
  - ISSUE-156
type: feature
ordinal: 166000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: aceite da ISSUE-071 (ressalva do usuário); refinada em 2026-11-17 após o parecer do Advisor na ISSUE-155 (plano ai-memory 01a1e987-1d45-7cc1-ac8a-4e3b2e72e65d). Situação: 'Rótulos da navegação' (lateral com rótulos ou só ícones acima de 64rem) vale só no documento e não é salva; o conf.toml existe mas não aceita preferências de interface. Com a ISSUE-156 a segunda barra e a opção empurrar/sobrepor (nav-layout) deixam de existir, então esta issue persiste SOMENTE as opções que sobrarem do painel de tema (hoje: rótulos da navegação) e outras que o usuário acordar explicitamente; não criar chave global para push/overlay e não reutilizar essas palavras com outro significado. Desenho adotado como ponto de partida (o usuário pode mudar na decisão do critério #1): estender o conf.toml reutilizando o SettingsStore; valor GLOBAL do servidor, sem sobrescrita local no navegador (coerente com o pedido de salvar no conf.toml); escrita global restrita à administração; o servidor aplica o valor ao renderizar (atributo no html, sem JS e sem piscar). A chave local antiga nav-layout fica inerte, sem limpeza destrutiva do armazenamento. Passos: 1) confirmar o escopo e a precedência, 2) leitura e escrita no conf.toml, 3) aplicar no head ao renderizar e remover o preview apenas no documento, 4) testes, docs de configuração e CHANGELOG. Bloqueada pela ISSUE-156.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Decisão registrada sobre o escopo (global do servidor, sem sobrescrita no navegador) e sobre quais opções do painel de tema são persistidas
- [ ] #2 Rótulos da navegação lidos do conf.toml e aplicados na renderização, sem JavaScript e sem piscar; a escrita fica restrita à administração
- [ ] #3 conf.toml existente sem as novas chaves continua válido, com valores padrão e validação de valores inválidos; nenhuma chave morta de empurrar/sobrepor
- [ ] #4 Testes cobrem leitura, padrão, valor inválido e aplicação no HTML; documentação atualizada
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-17T12:47:00Z

---

Refinada a pedido do usuário (2026-11-17): depende da ISSUE-156 (remove empurrar/sobrepor); escopo inicial global no conf.toml sem sobrescrita local, conforme recomendação do Advisor. A pergunta 3 do Advisor (global ou global com escolha por navegador) não foi respondida explicitamente; fica para o critério #1.

---
<!-- COMMENTS:END -->
