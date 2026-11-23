---
id: ISSUE-074
title: 074 — Remodelação responsiva da administração
status: Done
assignee:
  - '@ebrgr'
created_date: '2026-11-14T00:38:00Z'
updated_date: '2026-11-23T17:41:00Z'
labels:
  - admin
  - ui
milestone: m-10
dependencies:
  - ISSUE-071
references:
  - backlog/evidencias/issue-163/ESTUDO.md
  - backlog/evidencias/issue-074/AUDITORIA.md
priority: medium
type: feature
ordinal: 74000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Tarefa-mãe da remodelação responsiva da administração (pedido do usuário em 15/11/2026), no mesmo estilo já adotado na busca (ISSUE-107) e no painel de Tema (ISSUE-123). O escopo original (duas colunas, badge, /settings) foi absorvido: o selo de sessão saiu na ISSUE-109, tema e idioma no painel da ISSUE-110/123 e a navegação administrativa migrou da segunda barra da ISSUE-123 para o menu (+) na ISSUE-156.

Padrão visual comum: título da seção acima de um cartão com borda fina e cantos arredondados (bg-secondary, borda bg-tertiary, raio 8px); celular primeiro (uma coluna até 40rem, duas colunas a partir de 64rem quando couber); alvos de toque de 44px; rótulos acima dos campos; mensagens junto do cartão afetado; ações destrutivas numa zona de perigo separada e confirmada; sem JavaScript tudo funciona; com JavaScript as ações atualizam só o cartão (mecanismo da ISSUE-124); capturas em 390/768/1440 px nos temas Yuu e Pearl com aceite visual.

Páginas:
- ISSUE-125 — conta do administrador
- ISSUE-126 — gestão de perfis
- ISSUE-127 — configuração do servidor
- ISSUE-128 — login, revisões e mensagens

Relacionadas, no mesmo padrão: ISSUE-124 e ISSUE-108 (gestão de faixas, álbuns e artistas); ISSUE-083 (central de saúde) e ISSUE-112 (visualização de logs). O menu (+) da ISSUE-156 é o ponto de entrada para as ações administrativas.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 ISSUE-125 a ISSUE-128 concluídas no padrão visual comum
- [x] #2 Todas as páginas da administração funcionam sem JavaScript e em 390, 768 e 1440 px
- [x] #3 Capturas finais lado a lado nos temas Yuu e Pearl com aceite visual do conjunto
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

author: ebrgr
created: 2026-11-23T16:54:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-074` na branch `feat/issue-074-remodelacao-responsiva-da-administracao` criada a partir de `main`.

---

author: codex
created: 2026-11-23T17:09:00Z

---

Verificação: 216 aberturas de página ou downloads em 390/768/1440 px, Yuu e Pearl, com e sem JavaScript, em SQLite fictício e isolado. make check passou com 13/13 jornadas E2E e a referência regenerada coincidiu nos 190 arquivos. Parecer local: APROVADO.

- **Conta** — Identidade, senha e sessão em cartões. Zona de perigo em largura total. Estados normal, hover e pressed verificados nos botões primário e secundário.
- **Perfis e contas** — Lista e criação em cartões. A lista passa a duas colunas no desktop.
- **Perfil** — Identidade e avatar em uma coluna a 390 e 768 px e duas a 1440 px.
- **Gestão de conta** — Acesso, papel e zona de perigo separada.
- **Chaves de API** — O detalhe tinha 21–24 px. Passou a ter pelo menos 44 px. A revogação confirmada ficou num quadro separado. Renomear persistiu com POST nativo (303) e com troca do cartão (200, fragmento profile-keys).
- **Relay** — URL e token têm rótulos acima dos campos. Salvar, testar e remover ficam no cartão. Captura com configuração vazia.
- **Importação e exportação** — Cartões lado a lado a 1440 px e empilhados a 390 e 768 px. Exportar pelo link gerou o arquivo tunetrail_export.json. As jornadas de importação e exportação também passaram em make check.
- **Configuração do servidor** — Cartões por área, detalhes expansíveis e ações alcançáveis. A geração de parâmetros continua disponível com JavaScript. Captura com credenciais vazias.
- **Saúde da ingestão** — Atalhos de 36 px passaram a 44 px. Cartões ganharam borda fina e raio padrão. Inbox precede Outbox e as colunas ficam lado a lado a 1440 px. Descarte em cartão próprio.
- **Revisão de catálogo** — A revisão de mesclagem da faixa mostrou resumo, mudanças e zona de perigo, com leitura em uma coluna.
- **Revisão de exclusão** — Resumo e efeitos separados da confirmação de exclusão da conta e do perfil.
- **Login** — Formulário central com rótulos acima dos campos. Login de administrador e usuário comum exercitado na matriz.
- **Resgate** — Formulário central com instruções, rótulos e campos vazios.
- **Saída** — Página de confirmação e POST de saída exercitados. A página protegida retornou 401 após encerrar a sessão.
- **Erro 401** — Cartão de login após acesso anônimo à confirmação de saída.
- **Erro 403** — Cartão de acesso negado na configuração do servidor para usuário comum.
- **Erro 404** — Cartão de página indisponível ao revisar perfil inexistente.

O ajuste de formaction corrigiu o POST 405 com JavaScript e a edição de chave voltou a substituir só o cartão. As 102 capturas estão na auditoria, pendentes de aceite visual do conjunto (AC 3).

Proposta histórica: revisar a cadeia 163 → 160/161/162 → 158/159 → 100 → 074 e retirar a 074 como predecessora da 100. A 074 passaria a agregar a 163 e as filhas 125–128 depois da entrega delas. O grafo de dependências do Backlog permanece como está.

---

author: codex
created: 2026-11-23T17:26:00Z

---

Revisão após retorno do usuário. Removi o teto de .admin-shell e preservei o recuo da navegação. A 1440 px, a área útil mede 1302 px. Chaves e Relay passaram de 542 para 1302 px, e os pares de Conta, Perfil, Histórico e Configuração medem 643 px por cartão. A chave recém-criada quebra no próprio bloco monoespaçado. O botão Copiar fica abaixo e funciona após a atualização do cartão, com seleção manual se a área de transferência estiver indisponível. A regra compartilhada de hover e active passou em 1920 controles visíveis em 294 verificações, incluindo exceções por destino do relay, a confirmação da exceção e as revisões. Regeneradas 42 capturas das sete páginas solicitadas, com o valor da chave substituído na imagem. Removi o PARECER.md local. make check passou nas 13 jornadas E2E e a referência da interface coincidiu nos 190 arquivos. AC 3 aguarda o aceite visual do usuário.

---

author: claude
created: 2026-11-23T17:41:00Z

---

Aceite: aprovada pelo usuário em 23/11/2026, do jeito que está, após teste próprio da interface no servidor local com o banco dele. AC 3 (aceite visual do conjunto) dado pelo usuário. Parecer do Reviewer: APROVADO COM RESSALVAS (backlog/evidencias/issue-074/PARECER-REVIEWER.md). Riscos aceitos pelo usuário sem alteração: links de texto da administração ganham fundo no hover pela regra compartilhada; cartões sem teto de largura em telas muito largas (medido só até 1440 px); evidências com 7,7 MB no Git. Conflito conhecido com a branch da ISSUE-201 (enhance.js, CHANGELOG e referência da interface): resolver ao publicar, publicando a 201 primeiro e rebaseando esta sobre a main. Não publicada.

---
<!-- COMMENTS:END -->
