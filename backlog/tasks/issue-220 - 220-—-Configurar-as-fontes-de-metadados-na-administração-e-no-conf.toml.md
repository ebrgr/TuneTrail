---
id: ISSUE-220
title: 220 — Configurar as fontes de metadados na administração e no conf.toml
status: Aberta
assignee: []
created_date: '2026-11-24T19:53:00Z'
updated_date: '2026-11-24T19:54:00Z'
labels:
  - admin
  - config
milestone: m-13
dependencies:
  - ISSUE-214
references:
  - backlog/evidencias/issue-055/ESTUDO-COMPLEMENTAR.md
documentation:
  - doc-048
priority: medium
type: feature
ordinal: 233000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: decisão do usuário em 24/11/2026 (subsistema de enriquecimento compartilhado, começando pelas ISSUE-174 e 175) e o fluxo que ele descreveu: banco já salvo, Navidrome, MusicBrainz, depois ListenBrainz e os serviços comerciais. Card «Fontes de metadados» na administração do servidor: ligar e desligar cada provedor, ordem por cadeia, limite de requisições, timeout e teste de conexão. As chaves correspondentes ficam no conf.toml e nas variáveis de ambiente, no padrão da ISSUE-127 (cards, TOMLKey e ManagedSettingNames). Fontes comerciais vêm desligadas por padrão e são ligadas pelo servidor, e não por perfil; o ListenBrainz depende do relay aprovado do perfil. Coordenar com a ISSUE-213, que cuida do progresso, do histórico e dos limites na central de monitoramento: esta issue configura os provedores, aquela os acompanha.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Card com ligar, ordem, limite, timeout e teste de conexão por provedor, com hover e estados no padrão das telas já ajustadas
- [ ] #2 Cada campo do card tem a chave correspondente no conf.toml e na referência de configuração
- [ ] #3 Fontes comerciais desligadas por padrão
- [ ] #4 Escopo combinado com a ISSUE-213 sem duplicar a tela de monitoramento
<!-- AC:END -->
