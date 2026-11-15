---
id: ISSUE-123
title: 123 — Segunda barra lateral de administração
status: Aberta
assignee: []
created_date: '2026-11-15T01:19:00Z'
updated_date: '2026-11-15T01:19:00Z'
labels:
  - ui
  - admin
milestone: m-10
dependencies:
  - ISSUE-110
references:
  - backlog/evidencias/issue-110/advisor-segunda-barra.md
type: feature
ordinal: 132000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: pedido do usuário em 14/11/2026 e parecer do Advisor em backlog/evidencias/issue-110/advisor-segunda-barra.md. Ao clicar no ícone de configuração (Settings2), abre uma segunda barra de ícones ao lado da atual em telas largas e abaixo dela em telas estreitas. Itens: Entrar (visitante, /admin), Conta (/admin?section=account), Sair (POST com CSRF, só na administração), Perfis (/admin?section=profiles), Servidor (/admin/settings) e, por último, Sobre (versão e GitHub). Saúde (ISSUE-083) e Logs (ISSUE-112) entram quando existirem. Mecanismo: disclosure com aprimoramento progressivo (gatilho é link para /admin; JS compartilhado alterna hidden e aria-expanded; Esc fecha e devolve o foco); aberta pelo servidor nas páginas /admin; na prévia, só um aviso. Layout em telas largas: o usuário pediu as duas formas (empurrar o conteúdo, com --nav-w 58→116px, ou sobrepor) com uma opção para escolher; definir onde fica a escolha no início da tarefa. Coordenar com a ISSUE-071 (destino Mais no celular). Ícones Lucide novos: LogIn, LogOut, CircleUserRound ou UserRound, Server ou ServerCog.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Settings2 abre e fecha a segunda barra por clique e teclado; sem JS leva a /admin
- [ ] #2 Itens por papel: visitante vê Entrar e Sobre; administrador vê Conta, Sair (na administração), Perfis, Servidor e Sobre
- [ ] #3 Telas largas: empurrar ou sobrepor, conforme a opção escolhida; telas estreitas: abaixo da barra superior
- [ ] #4 Aberta nas páginas /admin; aviso na prévia
- [ ] #5 Traduções, testes e capturas desktop e celular com aceite visual
<!-- AC:END -->
