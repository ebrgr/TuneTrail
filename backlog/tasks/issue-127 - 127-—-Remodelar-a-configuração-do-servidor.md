---
id: ISSUE-127
title: 127 — Remodelar a configuração do servidor
status: Done
assignee:
  - '@ebrgr'
created_date: '2026-11-15T15:18:00Z'
updated_date: '2026-11-23T16:48:00Z'
labels:
  - ui
  - admin
milestone: m-10
dependencies:
  - ISSUE-124
references:
  - backlog/evidencias/issue-163/ESTUDO.md
  - backlog/evidencias/issue-127/ESTUDO.md
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
- [x] #1 Campos agrupados em cartões por área no padrão comum
- [x] #2 Origem do valor e necessidade de reinício visíveis sem poluir o campo
- [x] #3 Ações Salvar e Testar sempre alcançáveis; sem JavaScript o formulário funciona como hoje
- [ ] #4 Capturas 390/768/1440 em Yuu e Pearl com aceite visual
- [x] #5 Popup gera u/t/s do Navidrome no navegador sem enviar nem gravar a senha, com teste de conexão no próprio popup
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-20T18:33:00Z

---

Estudo da 163 (decisão do usuário em 20/11/2026): continua exclusivamente root; a 163 só retira o link indevido para o comum. Cartões, origem de valores e Salvar/Testar permanecem aqui.

---

author: ebrgr
created: 2026-11-23T14:17:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-127` na branch `feat/issue-127-remodelar-a-configuracao-do-servidor` criada a partir de `main`.

---

author: claude
created: 2026-11-23T14:23:00Z

---

Estudo 23/11/2026: popup <dialog> com cálculo md5(senha+sal) no navegador, aviso fixo (senha nunca gravada nem enviada; t+s equivalem a credencial), teste de conexão no popup, exibição de estado sem máscara. Auditoria de variáveis de ambiente em backlog/evidencias/issue-127/ESTUDO.md §7. Cartões, origem de valores e barra Salvar/Testar seguem pendentes (ACs 1–4).

---

author: claude
created: 2026-11-23T16:45:00Z

---

Aceite: aprovada pelo usuário em 23/11/2026, após teste próprio da interface no servidor local com o banco dele. Verificado: cartões por área (6) e ordem; origem do valor e aviso de reinício por cartão; barra Salvar/Testar fixa e formulário sem JavaScript preservado (testes de integração de cmd/api); popup do Navidrome com cálculo no navegador, teste de conexão no popup contra Navidrome 0.64.2; correspondência de 12 campos × 12 chaves do conf.toml provada por teste; make check verde. AC 4 deixado sem marca: o aceite visual foi do próprio usuário, mas não há capturas arquivadas em 390/768/1440 nos dois temas (só as do popup em backlog/evidencias/issue-127/). Derivadas abertas: ISSUE-197, 198, 199 e 200. O Reviewer não foi executado: o usuário aceitou direto de Em andamento. Não publicada.

---

author: ebrgr
created: 2026-11-23T16:48:00Z

---

Publicação: versão 0.69 (tag `v0.69`), branch `feat/issue-127-remodelar-a-configuracao-do-servidor` integrada ao `main` por fast-forward.

---
<!-- COMMENTS:END -->
