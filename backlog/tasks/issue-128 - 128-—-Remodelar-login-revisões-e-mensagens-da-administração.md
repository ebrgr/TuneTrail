---
id: ISSUE-128
title: 128 — Remodelar login, revisões e mensagens da administração
status: Aberta
assignee: []
created_date: '2026-11-15T15:18:00Z'
updated_date: '2026-11-22T20:44:00Z'
labels:
  - ui
  - admin
milestone: m-10
dependencies:
  - ISSUE-124
  - ISSUE-163
references:
  - backlog/evidencias/issue-163/ESTUDO.md
priority: medium
type: feature
ordinal: 141000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Parte da ISSUE-074 (remodelação responsiva da administração). Origem: pedido do usuário em 15/11/2026. Telas de apoio da administração: login (/admin sem sessão), revisão de mesclagem e exclusão do catálogo (catalog-review), revisão de exclusão de perfil (profile-review) e mensagens de erro e aviso. Proposta: login num cartão central no padrão comum; revisões com resumo do impacto em cartão, lista do que será alterado e confirmação na zona de perigo; erros e avisos com um único componente de mensagem, o mesmo dos demais cartões. Padrão visual comum (ISSUE-074): o mesmo estilo responsivo já usado na busca (ISSUE-107) e no painel de Tema (ISSUE-123) — título da seção acima de um cartão com borda fina e cantos arredondados (bg-secondary, borda bg-tertiary, raio 8px), celular primeiro (uma coluna até 40rem, duas colunas a partir de 64rem quando couber), alvos de toque de 44px, rótulos acima dos campos, mensagens de sucesso e erro junto do cartão afetado e ações destrutivas numa zona de perigo separada e confirmada. Funciona sem JavaScript; com JavaScript as ações atualizam só o cartão, usando o mecanismo da ISSUE-124. Capturas antes e depois em 390, 768 e 1440 px, nos temas Yuu e Pearl, com aceite visual do usuário.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Login em cartão no padrão comum
- [ ] #2 Revisões com resumo do impacto e confirmação na zona de perigo
- [ ] #3 Mensagens de erro e aviso com um componente único
- [ ] #4 Capturas 390/768/1440 em Yuu e Pearl com aceite visual
- [ ] #5 Login e resgate neutros para todos os papéis, com 401/403/404 conforme o estudo da 163 e erro uniforme
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-20T18:33:00Z

---

Reescopo (decisão do usuário em 20/11/2026, estudo da 163): login, resgate e estados 401/403/404 passam a atender todos os papéis, em /login com as entradas atuais preservadas, textos neutros e mensagens do estudo. Revisões ficam com escopo de dono ou root. Usa o retorno seguro da 163 e a confirmação de saída da 157.

---

author: claude
created: 2026-11-22T20:44:00Z

---

Início (22/11/2026): decisão de executar a 128 agora, em paralelo à fase 2 da 172, num agente Claude em worktree própria (para não competir pela cota do Codex). Fallback provisório do usuário comum até a ISSUE-125 criar /account: diretório público ou /admin?section=account; trocar para /account na 125. Cortes: A (login /login, resgate neutro, 401/403/404 e mensagem única), B (catalog-review e profile-review com resumo, lista e zona de perigo, escopo dono/root), C (capturas 390/768/1440 em Yuu e Pearl, referência visual, i18n, acessibilidade e sem JS). Reviewer ao final; Advisor só se tocar credenciais, bcrypt ou lock do resgate.

---
<!-- COMMENTS:END -->
