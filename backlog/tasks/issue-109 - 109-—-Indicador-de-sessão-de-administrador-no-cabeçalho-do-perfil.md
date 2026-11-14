---
id: ISSUE-109
title: 109 — Indicador de sessão de administrador no cabeçalho do perfil
status: Aberta
assignee: []
created_date: '2026-11-14T01:12:00Z'
labels:
  - ui
  - admin
milestone: m-10
dependencies:
  - ISSUE-106
priority: high
type: feature
ordinal: 109000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: pedido do usuário em 13/11/2026 e ROADMAP item 2 (badge de admin; absorve o critério correspondente da ISSUE-074).

Situação atual: o cabeçalho do perfil (`live.html`, `<header class="profile-header">`) mostra avatar, nome e "Listening profile · /slug", sem nenhum sinal de que a sessão de administrador está ativa. As páginas públicas nem sabem disso no servidor: o `live.js` descobre depois de carregar, com `fetch('/apis/web/v1/user')`, e só então acrescenta o botão de excluir escuta.

Investigação do ícone (Lucide, ISC, mesma licença de `icons/LICENSE`):

| Candidato | Leitura | Conflito no TuneTrail | Veredito |
| --- | --- | --- | --- |
| **ShieldCheck** | Acesso privilegiado, protegido | Nenhum | **Recomendado** |
| KeyRound | Tem uma chave | Confunde com as chaves de API dos perfis | Não |
| Crown | Dono, destaque | Confunde com o 1º lugar dos rankings | Não |
| UserCog / UserRoundCog | Configurações do usuário | Confunde com o ícone de configurações | Não |
| BadgeCheck | Conta verificada | Sugere perfil verificado, não sessão | Não |
| LockOpen | Desbloqueado | Sugere falta de segurança | Não |

Proposta: ShieldCheck pequeno (equivalente a 16px) como selo no canto inferior direito do avatar, com contorno na cor de fundo para separar da foto, cor de destaque do tema com contraste não textual de pelo menos 3:1 (WCAG 1.4.11). Texto acessível "Administrador conectado" (oculto visualmente) e `title` com a mesma frase. O selo é um link para `/admin`; o alvo de toque cobre o avatar (44×44px). Em telas largas, opcionalmente, um rótulo curto "Admin" ao lado do nome.

Passos:
1. Detecção no servidor: as páginas públicas leem a sessão de administrador pelo mesmo mecanismo da administração (cookie de sessão validado, sem criar sessão nova) e expõem `.Admin` ao template. Respostas com `Vary: Cookie` e cache privado (ISSUE-106).
2. Selo no cabeçalho do perfil; na página de seleção de perfis (sem cabeçalho), o sinal aparece no ícone de administração da barra lateral (ISSUE-110).
3. `live.js` usa `data-admin` no `<body>` em vez do `fetch` tardio; a exclusão rápida aparece sem atraso.
4. Traduções em `langs/en.json` e `langs/pt-BR.json`; ícone ShieldCheck acrescentado em `icons/`.

A ISSUE-108 reutiliza a detecção do passo 1 para mostrar a barra de ações.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Selo ShieldCheck no avatar do cabeçalho somente com sessão de administrador ativa
- [ ] #2 Visitantes nunca recebem o selo nem o atributo data-admin (testes)
- [ ] #3 Contraste do selo de pelo menos 3:1 em todos os temas; texto acessível e title traduzidos
- [ ] #4 live.js deixa de chamar /apis/web/v1/user para descobrir o papel
- [ ] #5 Respostas com Vary: Cookie e cache privado
- [ ] #6 Capturas nos temas Yuu e Pearl e aceite visual do usuário
<!-- AC:END -->
