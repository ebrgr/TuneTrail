---
id: ISSUE-123
title: 123 — Segunda barra lateral de administração
status: Aguardando aceite
assignee:
  - '@ebrgr'
created_date: '2026-11-15T01:19:00Z'
updated_date: '2026-11-15T02:00:00Z'
labels:
  - ui
  - admin
milestone: m-10
dependencies:
  - ISSUE-110
references:
  - backlog/evidencias/issue-110/advisor-segunda-barra.md
  - backlog/evidencias/issue-123/reviewer.md
  - backlog/evidencias/issue-123/segunda_barra.log
type: feature
ordinal: 132000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: pedido do usuário em 14/11/2026 e parecer do Advisor em backlog/evidencias/issue-110/advisor-segunda-barra.md. Ao clicar no ícone de configuração (Settings2), abre uma segunda barra de ícones ao lado da atual em telas largas e abaixo dela em telas estreitas. Itens: Entrar (visitante, /admin), Conta (/admin?section=account), Sair (POST com CSRF, só na administração), Perfis (/admin?section=profiles), Servidor (/admin/settings) e, por último, Sobre (versão e GitHub). Saúde (ISSUE-083) e Logs (ISSUE-112) entram quando existirem. Mecanismo: disclosure com aprimoramento progressivo (gatilho é link para /admin; JS compartilhado alterna hidden e aria-expanded; Esc fecha e devolve o foco); aberta pelo servidor nas páginas /admin; na prévia, só um aviso. Layout em telas largas: o usuário pediu as duas formas (empurrar o conteúdo, com --nav-w 58→116px, ou sobrepor) com uma opção para escolher; definir onde fica a escolha no início da tarefa. Coordenar com a ISSUE-071 (destino Mais no celular). Ícones Lucide novos: LogIn, LogOut, CircleUserRound ou UserRound, Server ou ServerCog.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Settings2 abre e fecha a segunda barra por clique e teclado; sem JS leva a /admin
- [x] #2 Itens por papel: visitante vê Entrar e Sobre; administrador vê Conta (com Sair na própria página), Perfis, Servidor e Sobre
- [x] #3 Telas largas: empurrar ou sobrepor, conforme a opção escolhida; telas estreitas: abaixo da barra superior
- [x] #4 Aberta nas páginas /admin; aviso na prévia
- [ ] #5 Traduções, testes e capturas desktop e celular com aceite visual
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: ebrgr
created: 2026-11-15T01:45:00Z

---

Início: branch `feat/issue-123-segunda-barra-lateral-de-administracao` criada a partir de `main`.

---

author: claude
created: 2026-11-15T01:46:00Z

---

Decisões em 14/11/2026: (1) usuário — a opção empurrar/sobrepor fica no painel Tema; o painel separa cores, esta opção e o idioma em blocos com borda fina e cantos arredondados no estilo do projeto. (2) implementação — sem item Sair na barra: o logout exige token de envio de uso único por formulário (consumeSubmission), e um formulário em todas as páginas consumiria o token dos demais; Conta leva a /admin?section=account, onde Sair já existe. Critério #2 ajustado.

---

author: ebrgr
created: 2026-11-15T01:54:00Z

---

Verificação: make check 12/12 e referência sem diferenças; validador real Chrome/Firefox/HTTPS PASS; roteiro segunda_barra.py PASS (clique, Espaço, Esc com foco, empurrar desloca 58 px e sobrepor não, preferência lembrada, aberta na administração com Servidor marcado, faixa abaixo da barra no celular); testes de papel, estado aberto e erro 429 antes do login; Reviewer com os dois itens obrigatórios corrigidos. Falta o aceite visual (capturas em backlog/evidencias/issue-123/).

---

author: claude
created: 2026-11-15T02:00:00Z

---

Ajustes pedidos pelo usuário antes do aceite: (1) no modo sobrepor a barra começa fechada (também nas páginas da administração, sem piscar) e fecha depois do clique num item; (2) ícone de Conta trocado de CircleUserRound para LockKeyhole (cadeado), para não lembrar o perfil nem confundir com as chaves de API. Roteiro segunda_barra.py, make check e validador de cache aprovados.

---
<!-- COMMENTS:END -->
