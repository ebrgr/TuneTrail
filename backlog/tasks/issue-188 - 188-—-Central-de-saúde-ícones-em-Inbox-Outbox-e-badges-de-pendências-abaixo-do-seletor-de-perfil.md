---
id: ISSUE-188
title: >-
  188 — Central de saúde: ícones em Inbox/Outbox e badges de pendências abaixo
  do seletor de perfil
status: Em andamento
assignee:
  - '@ebrgr'
created_date: '2026-11-22T19:36:00Z'
updated_date: '2026-11-22T19:46:00Z'
labels:
  - ui
  - ingestao
milestone: m-13
dependencies: []
references:
  - backlog/evidencias/issue-188/manifest.json
type: feature
ordinal: 201000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: pedido do usuário em 22/11/2026. (1) Em /admin/health, colocar um ícone antes dos títulos Inbox e Outbox, no mesmo estilo dos ícones do app (família Lucide já usada na navegação, ex.: HeartPulse, UserRoundCog); escolher ícones coerentes com o sentido (entrada/saída) e reutilizar o helper de ícones existente, sem SVG avulso. (2) Abaixo do seletor de perfil, mostrar um resumo das pendências em aberto como badges (ou caixas de texto compactas) que seguem o mesmo padrão de cores da UI (tokens existentes de sucesso/aviso/erro/neutro, claro e escuro). Interpretação a confirmar com o usuário: 'tarefas em aberto' = itens de saúde que pedem ação no perfil visitado: revisões pendentes (hoje no atalho 'Pendências de revisão (N)'), falhas permanentes, recebimentos com metadados faltantes ou incompletos na janela do Inbox, e itens do Outbox ainda pendentes ou em retry; badge zerada em tom neutro, com contagem e link âncora para a seção correspondente; os números vêm das mesmas consultas já usadas pelas seções (sem consulta nova pesada). Regras de CSS do projeto: só min-width 40/64/90rem, arquivos de CSS com até 400 linhas, regenerar com make ui.reference; textos com tr e chaves de tradução; acessibilidade (nome acessível nas badges, contraste). Não altera comportamento da ingestão.
<!-- SECTION:DESCRIPTION:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-22T19:38:00Z

---

Decisão do usuário (22/11/2026): combinação A — ícones Lucide Inbox (Inbox) e Send (Outbox). Ícones opcionais nas badges: ListChecks (revisões, aviso), CircleAlert (falhas permanentes, erro), TriangleAlert (metadados, aviso), RefreshCw (retry do outbox), Check (zerado). Interpretação das badges mantida como descrita; Reviewer após a entrega.

---

author: ebrgr
created: 2026-11-22T19:38:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-188` na branch `feat/issue-188-central-de-saude-icones-em` criada a partir de `main`.

---

author: codex
created: 2026-11-22T19:43:00Z

---

Verificação: /admin/health exibe ícones decorativos nos títulos e quatro badges com links para as seções. Os totais usam dados já carregados do perfil, e a badge de metadados acompanha a janela e o filtro de chave do Inbox. Testes dos pacotes afetados com TUNETRAIL_WEB_DIR apontando para assets passaram, cobrindo contagens, zeros neutros, isolamento entre perfis e ícones. A referência de UI foi regenerada e make check passou (13/13 jornadas E2E). Capturas em Chromium nos temas Yuu e Pearl verificaram contraste mínimo de 6,38:1, foco e navegação por teclado.

---

author: claude
created: 2026-11-22T19:46:00Z

---

Reviewer: APROVADO, sem BUG. Riscos baixos: atalho 'Pendências de revisão' duplica a badge; semânticas de janela diferentes entre badges; ícones dos h2 pequenos. Testes adminui/webassets/webui/engine ok.

---
<!-- COMMENTS:END -->
