---
id: ISSUE-188
title: >-
  188 — Central de saúde: ícones em Inbox/Outbox e badges de pendências abaixo
  do seletor de perfil
status: Aberta
assignee: []
created_date: '2026-11-22T19:36:00Z'
labels:
  - ui
  - ingestao
milestone: m-13
dependencies: []
type: feature
ordinal: 201000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: pedido do usuário em 22/11/2026. (1) Em /admin/health, colocar um ícone antes dos títulos Inbox e Outbox, no mesmo estilo dos ícones do app (família Lucide já usada na navegação, ex.: HeartPulse, UserRoundCog); escolher ícones coerentes com o sentido (entrada/saída) e reutilizar o helper de ícones existente, sem SVG avulso. (2) Abaixo do seletor de perfil, mostrar um resumo das pendências em aberto como badges (ou caixas de texto compactas) que seguem o mesmo padrão de cores da UI (tokens existentes de sucesso/aviso/erro/neutro, claro e escuro). Interpretação a confirmar com o usuário: 'tarefas em aberto' = itens de saúde que pedem ação no perfil visitado: revisões pendentes (hoje no atalho 'Pendências de revisão (N)'), falhas permanentes, recebimentos com metadados faltantes ou incompletos na janela do Inbox, e itens do Outbox ainda pendentes ou em retry; badge zerada em tom neutro, com contagem e link âncora para a seção correspondente; os números vêm das mesmas consultas já usadas pelas seções (sem consulta nova pesada). Regras de CSS do projeto: só min-width 40/64/90rem, arquivos de CSS com até 400 linhas, regenerar com make ui.reference; textos com tr e chaves de tradução; acessibilidade (nome acessível nas badges, contraste). Não altera comportamento da ingestão.
<!-- SECTION:DESCRIPTION:END -->
