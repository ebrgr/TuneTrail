---
id: ISSUE-127
title: 127 — Remodelar a configuração do servidor
status: Aberta
assignee: []
created_date: '2026-11-15T15:18:00Z'
updated_date: '2026-11-15T15:18:00Z'
labels:
  - ui
  - admin
milestone: m-10
dependencies:
  - ISSUE-124
priority: medium
type: feature
ordinal: 140000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Parte da ISSUE-074 (remodelação responsiva da administração). Origem: pedido do usuário em 15/11/2026. Página /admin/settings (ícone Servidor da segunda barra). Hoje: lista longa de campos, cada um num bloco com valores salvo, efetivo e em execução e aviso de variável de ambiente. Proposta: cartões por área (Geral e log, Horário, MusicBrainz, Imagens, Subsonic, Rede); cada campo mostra a origem do valor (padrão, salvo, ambiente) de forma compacta; aviso de reinício no cartão afetado; barra de ações Salvar e Testar sempre alcançável no rodapé; campos definidos por variável de ambiente claramente bloqueados. Padrão visual comum (ISSUE-074): o mesmo estilo responsivo já usado na busca (ISSUE-107) e no painel de Tema (ISSUE-123) — título da seção acima de um cartão com borda fina e cantos arredondados (bg-secondary, borda bg-tertiary, raio 8px), celular primeiro (uma coluna até 40rem, duas colunas a partir de 64rem quando couber), alvos de toque de 44px, rótulos acima dos campos, mensagens de sucesso e erro junto do cartão afetado e ações destrutivas numa zona de perigo separada e confirmada. Funciona sem JavaScript; com JavaScript as ações atualizam só o cartão, usando o mecanismo da ISSUE-124. Capturas antes e depois em 390, 768 e 1440 px, nos temas Yuu e Pearl, com aceite visual do usuário.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Campos agrupados em cartões por área no padrão comum
- [ ] #2 Origem do valor e necessidade de reinício visíveis sem poluir o campo
- [ ] #3 Ações Salvar e Testar sempre alcançáveis; sem JavaScript o formulário funciona como hoje
- [ ] #4 Capturas 390/768/1440 em Yuu e Pearl com aceite visual
<!-- AC:END -->
