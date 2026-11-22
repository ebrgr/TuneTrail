---
id: ISSUE-125
title: 125 — Remodelar a página de conta do administrador
status: Em andamento
assignee:
  - '@ebrgr'
created_date: '2026-11-15T15:18:00Z'
updated_date: '2026-11-22T21:03:00Z'
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

author: ebrgr
created: 2026-11-22T20:45:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-125` na branch `feat/issue-125-remodelar-a-pagina-de-conta` criada a partir de `main`.

---

author: claude
created: 2026-11-22T20:58:00Z

---

Implementação (22/11/2026): GET /account para qualquer sessão (comum, dono, root) com cartões Identidade, Senha (aviso antes de enviar, requisitos, erros por campo com role=alert/aria-describedby), Sessão (Sair… secundário via /admin/sign-out) e zona de perigo reservada vazia. /admin?section=account redireciona 303 para /account (após autorização). AccountDestination=/account (menu, atalhos, fallback pós-login, tela de confirmação de saída). Sem JS: POST completo com erro no cartão; com JS: fragmento do cartão (mecanismo da 124, X-TuneTrail-Fragment/Submission). 'account' virou slug reservado. Credenciais, bcrypt, sessões e lock do resgate não foram tocados (ChangeAccount reutilizado). Verificação: make check verde (go test, ui.reference.check com ISSUE-075/reference regenerada, e2e 13/13); testes novos em engine/account_issue125_test.go. Capturas 390/768/1440: dispensadas por decisão do usuário (ele testa no servidor); critério #4 segue pendente de aceite visual.

---

author: claude
created: 2026-11-22T21:03:00Z

---

Reviewer: APROVADO, sem BUG. Riscos baixos: slug 'account' reservado, POST /account sem restrição de método, falta mensagem 'senha alterada' pós-troca (ISSUE-128), /account com query dá 400. Testes adminui/adminweb/webassets/webui/admin/engine ok. Aceite visual do usuário no servidor (sem capturas, por decisão dele).

---
<!-- COMMENTS:END -->
