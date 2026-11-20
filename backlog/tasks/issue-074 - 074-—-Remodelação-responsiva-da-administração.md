---
id: ISSUE-074
title: 074 — Remodelação responsiva da administração
status: Aberta
assignee: []
created_date: '2026-11-14T00:38:00Z'
updated_date: '2026-11-20T18:33:00Z'
labels:
  - admin
  - ui
milestone: m-10
dependencies:
  - ISSUE-071
references:
  - backlog/evidencias/issue-163/ESTUDO.md
priority: medium
type: feature
ordinal: 74000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Tarefa-mãe da remodelação responsiva da administração (pedido do usuário em 15/11/2026), no mesmo estilo já adotado na busca (ISSUE-107) e no painel de Tema (ISSUE-123). O escopo original (duas colunas, badge, /settings) foi absorvido: o selo de sessão saiu na ISSUE-109, tema e idioma no painel da ISSUE-110/123 e a navegação de seções na segunda barra da ISSUE-123.

Padrão visual comum: título da seção acima de um cartão com borda fina e cantos arredondados (bg-secondary, borda bg-tertiary, raio 8px); celular primeiro (uma coluna até 40rem, duas colunas a partir de 64rem quando couber); alvos de toque de 44px; rótulos acima dos campos; mensagens junto do cartão afetado; ações destrutivas numa zona de perigo separada e confirmada; sem JavaScript tudo funciona; com JavaScript as ações atualizam só o cartão (mecanismo da ISSUE-124); capturas em 390/768/1440 px nos temas Yuu e Pearl com aceite visual.

Páginas:
- ISSUE-125 — conta do administrador
- ISSUE-126 — gestão de perfis
- ISSUE-127 — configuração do servidor
- ISSUE-128 — login, revisões e mensagens

Relacionadas, no mesmo padrão: ISSUE-124 e ISSUE-108 (gestão de faixas, álbuns e artistas); ISSUE-083 (central de saúde) e ISSUE-112 (visualização de logs), que entram como itens novos da segunda barra.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 ISSUE-125 a ISSUE-128 concluídas no padrão visual comum
- [ ] #2 Todas as páginas da administração funcionam sem JavaScript e em 390, 768 e 1440 px
- [ ] #3 Capturas finais lado a lado nos temas Yuu e Pearl com aceite visual do conjunto
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-14T01:06:00Z

---

Exclusão, mesclagem e edição de itens de mídia (faixa, álbum, artista) saem da administração geral e passam a ser feitas pela barra de ações da página do item (ISSUE-108). Esta tarefa continua cuidando do layout da administração, do badge e de /settings.

---

author: claude
created: 2026-11-14T01:12:00Z

---

Badge de administrador (critério 2 e passo 3) movido para a ISSUE-109, que cria o selo ShieldCheck no cabeçalho do perfil com detecção da sessão no servidor. O acesso à administração pela barra lateral passa a ser o submenu da ISSUE-110; esta tarefa continua com o layout da administração, a zona de perigo e /settings.

---

author: claude
created: 2026-11-15T15:18:00Z

---

Reescrita como tarefa-mãe por pedido do usuário em 15/11/2026: as páginas da administração ganham tarefas próprias (ISSUE-125 a 128) no mesmo estilo responsivo; o escopo antigo foi absorvido pelas ISSUE-109, 110 e 123.

---

author: claude
created: 2026-11-20T18:33:00Z

---

Estudo da 163 (20/11/2026): atualizar as referências históricas à segunda barra conforme a 156 e usar os mesmos cartões e zonas destrutivas nas 125–128. A 074 não deve ser predecessora das filhas que agrega; revisar o vínculo histórico 163 → 160/161/162 → 158/159 → 100 → 074 ao ajustar dependências.

---
<!-- COMMENTS:END -->
