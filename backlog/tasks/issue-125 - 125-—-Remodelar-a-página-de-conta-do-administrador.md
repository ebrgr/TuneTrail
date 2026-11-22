---
id: ISSUE-125
title: 125 — Remodelar a página de conta do administrador
status: Aberta
assignee: []
created_date: '2026-11-15T15:18:00Z'
updated_date: '2026-11-22T22:17:00Z'
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
ordinal: 138000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Parte da ISSUE-074 (remodelação responsiva da administração). Origem: pedido do usuário em 15/11/2026. Página /admin?section=account (ícone Conta da segunda barra, ISSUE-123). Hoje: título Administração, link de configuração, botão Sair solto no topo e um único bloco com nome de usuário e troca de senha. Proposta: cartão Identidade (nome de usuário), cartão Senha (nova senha e confirmação, requisitos visíveis), cartão Sessão (sessão atual e Sair com peso visual secundário) e zona de perigo reservada para ações futuras. Padrão visual comum (ISSUE-074): o mesmo estilo responsivo já usado na busca (ISSUE-107) e no painel de Tema (ISSUE-123) — título da seção acima de um cartão com borda fina e cantos arredondados (bg-secondary, borda bg-tertiary, raio 8px), celular primeiro (uma coluna até 40rem, duas colunas a partir de 64rem quando couber), alvos de toque de 44px, rótulos acima dos campos, mensagens de sucesso e erro junto do cartão afetado e ações destrutivas numa zona de perigo separada e confirmada. Funciona sem JavaScript; com JavaScript as ações atualizam só o cartão, usando o mecanismo da ISSUE-124. Capturas antes e depois em 390, 768 e 1440 px, nos temas Yuu e Pearl, com aceite visual do usuário.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Cartões Identidade, Senha e Sessão no padrão comum, em uma ou duas colunas conforme a largura
- [ ] #2 Sair com peso secundário; mensagens junto do cartão
- [ ] #3 Funciona sem JavaScript; com JavaScript atualiza só o cartão
- [ ] #4 Capturas 390/768/1440 em Yuu e Pearl com aceite visual
- [ ] #5 Conta acessível a qualquer sessão autenticada em /account, sem cair no 403 do usuário comum
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-20T18:33:00Z

---

Reescopo (decisão do usuário em 20/11/2026, estudo da 163): a página Conta passa a ser destino de qualquer sessão (comum e root) em /account, com cartões Identidade, Senha e Sessão, conforme doc-041 §7. A troca de senha encerra todas as sessões e avisa antes; username separado de slug. A 163 define o destino e a navegação; a 157 fornece o Sair….

---

author: claude
created: 2026-11-22T20:45:00Z

---

Início (22/11/2026): decisão do usuário de fazer a 125 antes da 128, pois ela cria /account (destino pós-login do usuário comum). Executada por um agente Claude em worktree própria (Codex ocupado com a fase 2 da 172). Reviewer ao final; Advisor só se tocar credenciais, bcrypt, sessões ou lock do resgate (a troca de senha já revoga as sessões: manter o contrato, não reescrever).

---

author: claude
created: 2026-11-22T20:48:00Z

---

Decisão do usuário (22/11/2026): sem capturas de tela nesta issue; o aceite visual será feito por ele no servidor. O critério #4 passa a ser verificado pelo usuário.

---

author: claude
created: 2026-11-22T21:15:00Z

---

Teste do usuário no servidor (22/11/2026): resgate de token em /admin/redeem mostrou 'Sua conta não tem permissão para acessar esta página.' e a página não orienta a senha. Diagnóstico: falha de CSRF/preauth no POST é exibida com a mensagem genérica de 403 (reproduzido com token CSRF inválido); fluxo normal funciona. Causa provável: cookies sem separação por porta em 127.0.0.1 (instância 4110 e 4125). Ajuste incluído na 125: mensagem de página expirada, regras da senha e confirmação no resgate, sucesso leva ao login com aviso.

---

author: claude
created: 2026-11-22T21:20:00Z

---

Requisito do usuário (22/11/2026): todas as telas tocadas pela 125 seguem o padrão de cores e os mesmos estados de hover, active e foco do restante do app (tokens de tokens.css, button.large-button, a:hover, focus-visible global, .admin-sections), nos temas Yuu e Pearl, sem cor fixa nova.

---

author: claude
created: 2026-11-22T22:13:00Z

---

Pedido do usuário (22/11/2026): a página de resgate deve ter uma caixa explícita orientando a sair do perfil root e abrir 'Todos os perfis' (/) antes de resgatar; com sessão aberta, versão enfática com Sair…. Perguntas em aberto levadas ao usuário: zona de perigo vazia na /account (recomendado remover até existir ação) e seletor de ícones do avatar (hoje campo de texto com 5 nomes; mover para a ISSUE-126 ou corrigir já).

---

author: claude
created: 2026-11-22T22:17:00Z

---

Decisões do usuário (22/11/2026): manter a zona de perigo vazia em /account; trocar já o campo de texto de ícone por dropdown (5 ícones + iniciais) na criação do perfil e no avatar, dentro da 125. A separação de 'Meu perfil' e 'Minha conta' segue na ISSUE-126.

---
<!-- COMMENTS:END -->
