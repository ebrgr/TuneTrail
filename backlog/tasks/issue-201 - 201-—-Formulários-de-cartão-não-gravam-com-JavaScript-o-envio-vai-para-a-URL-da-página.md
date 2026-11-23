---
id: ISSUE-201
title: >-
  201 — Formulários de cartão não gravam com JavaScript: o envio vai para a URL
  da página
status: Done
assignee:
  - '@ebrgr'
created_date: '2026-11-23T17:04:00Z'
updated_date: '2026-11-23T18:21:00Z'
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

author: ebrgr
created: 2026-11-23T17:13:00Z

---

Verificação: causa em enhance.js:50 (submitter.formAction devolve a URL do documento sem o atributo formaction), corrigida com getAttribute. Teste de regressão em internal/webui/enhance.test.mjs (3 casos). Verificado no navegador, instância isolada: avatar (ícone), nome do perfil, criação de chave, relay (via formaction do botão) e usuário da conta gravam, com POST nas actions certas e valores no banco; senha da conta e formulários de gestão de mídia não foram exercitados (mesmo caminho de código). make check verde, referência da interface regenerada. Aguarda o aceite do usuário.

---

author: claude
created: 2026-11-23T17:41:00Z

---

Aceite: aprovada pelo usuário em 23/11/2026, após teste próprio no servidor local com o banco dele (branch da ISSUE-074, que carrega a mesma correção). Verificado por mim em instância isolada: avatar (ícone), nome do perfil, criação de chave, relay (pelo formaction do botão) e usuário da conta gravam, com POST na action certa e valor no banco; teste de regressão em internal/webui/enhance.test.mjs; make check verde. AC 1 deixado sem marca: troca de senha da conta e formulários da gestão de mídia não foram exercitados no navegador (mesmo caminho de código). Não publicada. Conflito conhecido com a branch da ISSUE-074 (enhance.js, CHANGELOG e referência da interface), a resolver ao publicar.

---

author: ebrgr
created: 2026-11-23T18:21:00Z

---

Publicação: versão 0.70 (tag `v0.70`), branch `fix/issue-201-formularios-de-cartao-nao-gravam` integrada ao `main` por fast-forward.

---
<!-- COMMENTS:END -->
