---
id: ISSUE-153
title: 153 — Persistir as opções do painel de tema no conf.toml
status: Aberta
assignee: []
created_date: '2026-11-16T18:57:00Z'
updated_date: '2026-11-16T18:57:00Z'
labels:
  - config
  - ui
milestone: m-10
dependencies:
  - ISSUE-071
type: feature
ordinal: 166000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: aceite da ISSUE-071 (ressalva do usuário). As opções do painel de tema não são salvas em lugar nenhum além do navegador: 'Rótulos da navegação' (lateral com rótulos ou só ícones acima de 64rem) vale só no documento, e empurrar/sobrepor (ISSUE-123) fica no armazenamento do navegador. O usuário quer considerar guardá-las no conf.toml. Situação: o conf.toml existe, mas não aceita preferências de UI. Recomendação do Codex: estender o conf.toml reutilizando o SettingsStore; alternativas: arquivo novo ou banco/variáveis de ambiente. Atenção: o valor no conf.toml é global do servidor (todos os perfis e navegadores); decidir se o navegador ainda pode sobrescrever. O servidor deve aplicar o valor ao renderizar, sem JS e sem piscar. Passos: 1) Advisor decide o desenho (escopo global x por navegador, precedência conf.toml x navegador, validação e valores padrão, compatibilidade com conf.toml existente). 2) Implementar leitura e escrita. 3) Aplicar no head ao renderizar. 4) Testes, docs de configuração e CHANGELOG.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Decisão registrada sobre o escopo (global do servidor) e a precedência entre conf.toml e navegador
- [ ] #2 Rótulos da navegação e modo empurrar/sobrepor lidos do conf.toml e aplicados na renderização, sem JS e sem piscar
- [ ] #3 conf.toml existente sem as novas chaves continua válido, com valores padrão e validação de valores inválidos
- [ ] #4 Testes cobrem leitura, padrão, valor inválido e aplicação no HTML; documentação atualizada
<!-- AC:END -->
