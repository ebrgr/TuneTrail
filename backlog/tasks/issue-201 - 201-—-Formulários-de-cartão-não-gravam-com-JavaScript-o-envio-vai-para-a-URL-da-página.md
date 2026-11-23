---
id: ISSUE-201
title: >-
  201 — Formulários de cartão não gravam com JavaScript: o envio vai para a URL
  da página
status: Em andamento
assignee:
  - '@ebrgr'
created_date: '2026-11-23T17:04:00Z'
updated_date: '2026-11-23T17:10:00Z'
labels:
  - ui
  - admin
milestone: m-10
dependencies: []
references:
  - backlog/evidencias/issue-127/ESTUDO.md
priority: high
type: bug
ordinal: 214000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: teste do usuário em 23/11/2026 («os avatares não ficam salvos depois de escolhidos») e reprodução em instância isolada. Com JavaScript ligado, os formulários marcados com data-enhance enviam o POST para a URL da própria página (por exemplo POST /account/profile) em vez da action do formulário (/admin/profile/N/avatar), e o servidor responde 405. O enhance.js então recarrega a página sem aviso, e nada é gravado: avatar (ícone e foto), nome do perfil, e os demais cartões das abas Conta, Perfil, Chaves de API e Relay, além da gestão de mídia. Causa: assets/js/enhance.js:50 usa event.submitter.formAction; quando o botão não tem o atributo formaction, o navegador devolve a URL do documento e não a action do formulário. Introduzido na ISSUE-126 (e8c74ab, v0.64) para o Salvar e Testar do relay. Sem JavaScript o formulário funciona. Correção: ler o atributo formaction do botão (getAttribute) e, na ausência, a action do formulário. Verificado em 23/11/2026: avatar_icon continua vazio e display_name continua igual depois de Salvar, com POST 405 na rede.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Com JavaScript ligado, avatar (ícone e foto), nome do perfil, chaves de API, relay, senha e usuário da conta gravam, e a gestão de mídia continua gravando
- [x] #2 A resolução da URL de envio do enhance.js usa o formaction do botão quando existe e a action do formulário quando não, com teste de regressão
- [x] #3 Verificado no navegador, contra instância isolada, que a rede mostra POST na action do formulário e o banco guarda o valor
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: ebrgr
created: 2026-11-23T17:04:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-201` na branch `fix/issue-201-formularios-de-cartao-nao-gravam` criada a partir de `main`.

---
<!-- COMMENTS:END -->
