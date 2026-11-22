---
id: ISSUE-125
title: 125 — Remodelar a página de conta do administrador
status: Em andamento
assignee:
  - '@ebrgr'
created_date: '2026-11-15T15:18:00Z'
updated_date: '2026-11-22T22:16:00Z'
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

author: claude
created: 2026-11-22T21:24:00Z

---

Ajustes pós-teste (22/11/2026): (1) resgate de token: regras da senha visíveis antes do envio (mín. 8 caracteres, máx. 72 bytes, como validAccountPassword), campo Confirmar senha, erros por campo, sucesso redireciona para /admin?redeemed=1 (login com 'Senha definida…'); CSRF/preauth inválido em resgate e login mostra 'Esta página expirou ou foi aberta em outra aba' com formulário novo (403, username preservado, nunca token/senha), sem afrouxar CSRF/SameOrigin. (2) estilos de /account e resgate reutilizam tokens e estados globais (foco global, hover/active/disabled, sem cor fixa); verificação em scripts/check_web_style.py com testes. make check verde. Sem capturas (decisão do usuário).

---

author: claude
created: 2026-11-22T21:28:00Z

---

Reviewer (reauditoria): APROVADO, sem BUG. Riscos baixos: leitura prévia do corpo para preservar o username, hover do botão primário derivado por color-mix, anel de foco da administração (2px primário), texto de senha da /account diz caracteres e a regra é bytes, tabela de mensagens realinhada. Testes ok.

---

author: claude
created: 2026-11-22T21:52:00Z

---

Causa raiz do erro do usuário em /admin/redeem (22/11/2026): a página de resgate declarava Referrer-Policy: no-referrer; sob essa política os navegadores enviam Origin: null no POST do formulário, e a checagem same-origin recusava com a mensagem de permissão (curl e testes não reproduziam porque enviavam Origin correto). Correção: Referrer-Policy same-origin na página de resgate (sem segredo na URL; query é recusada), recusa de origem no resgate mostra a mensagem de página expirada com formulário novo, teste de regressão com Origin: null. make check verde, 13/13 E2E.

---

author: claude
created: 2026-11-22T22:16:00Z

---

Resgate (22/11/2026): caixa de aviso no topo de /admin/redeem (role=note, estilo .account-warning) com links reais Sair… (/admin/sign-out) e Todos os perfis (/). Sem sessão: versão informativa; com sessão (SessionIdentity ok): versão enfática com o username da própria sessão. CSRF/origem e regras de senha inalterados. Testes em engine/account_issue125_test.go (visitante, root, comum, pt-BR, sem eco de token/senha). make check verde.

---
<!-- COMMENTS:END -->
