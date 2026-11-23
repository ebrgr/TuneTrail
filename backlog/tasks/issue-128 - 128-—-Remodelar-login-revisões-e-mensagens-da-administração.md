---
id: ISSUE-128
title: 128 — Remodelar login, revisões e mensagens da administração
status: Done
assignee:
  - '@ebrgr'
created_date: '2026-11-15T15:18:00Z'
updated_date: '2026-11-23T01:45:00Z'
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

author: claude
created: 2026-11-22T20:45:00Z

---

Suspensa pelo usuário em 22/11/2026 antes de qualquer código: fazer a ISSUE-125 (conta) primeiro, porque ela cria /account, destino pós-login do usuário comum que a 128 usa. Worktree preservada (feat/issue-128-remodelar-login-revisoes-e-mensagens, sem mudanças). Retomar depois da 125, trocando o fallback provisório para /account.

---

author: claude
created: 2026-11-23T00:45:00Z

---

Reinício (22/11/2026): a worktree antiga estava 49 commits atrás da main e foi recriada do zero a partir da v0.65 (sem perda: não tinha trabalho). REESCOPO depois das 125 e 126: já entregue pela 125 o resgate de token (regras da senha, confirmação, caixa de orientação com Sair… e Todos os perfis, mensagem de página expirada, Referrer-Policy same-origin) e o destino pós-login de qualquer papel em /account; já entregue pela 126 a conta única com abas e a Administração (lista de contas). Resta: (1) login em cartão central e rota neutra GET /login para todos os papéis, mantendo /admin/login e os contratos de formulário, com retorno seguro e sessão expirada com mensagem neutra; (2) telas 401/403/404 no mesmo shell neutro, com as mensagens do estudo da 163 e sem revelar se o 404 foi permissão ou ausência; (3) um único componente de mensagem de erro, aviso e sucesso usado em todos os cartões (login, resgate, conta, abas, revisões); (4) catalog-review, profile-review e profile-delete em cartões: resumo do impacto, lista do que será alterado e confirmação em zona de perigo, escopo dono ou root; (5) padrão de cores e estados do app (sem cor fixa). Sem capturas de tela; o usuário testa no servidor.

---

author: ebrgr
created: 2026-11-23T00:46:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-128` na branch `feat/issue-128-remodelar-login-revisoes-e-mensagens` criada a partir de `main`.

---

author: claude
created: 2026-11-23T01:12:00Z

---

Verificação (23/11/2026, agente claude, worktree issue-128, base v0.65): cortes A–D implementados em 3 commits; make check verde (go vet, go test ./..., ui.reference.check com a referência regenerada, e2e 13/13). A) GET /login para todos os papéis (/admin/login e contratos POST mantidos; sessão aberta vai ao destino validado; cookie expirado e páginas protegidas sem sessão mostram 'Entre para continuar.'; retorno seguro com fallback /account e /admin), cartão central, estado neutro 401/403/404 com as mensagens do estudo e os links da própria sessão (404 idêntico para perfil protegido e ausente), 'Senha alterada' após a troca de senha, slug login reservado. B) componente admin/message (role=alert erro, role=status aviso/sucesso, ids para aria-describedby) em login, resgate, conta, abas, configurações e gestão. C) catalog-review, profile-review e profile-delete em cartões (resumo, lista, zona de perigo), na página e no diálogo; escopo: catálogo e exclusão de conta = root, apagar escutas = dono ou root. D) i18n en/pt-BR, troca de idioma permanece em /login, contraste do erro no Pearl, alvos de 44px. Testes novos: engine/{login,messages,reviews}_issue128_test.go e internal/adminweb/language_return_128_test.go. Sem capturas (o usuário testa no servidor). Registro: backlog/evidencias/issue-128/RECONCILIACAO.md. Pendente: aceite visual do usuário; Reviewer pelo despachante.

---

author: claude
created: 2026-11-23T01:40:00Z

---

Reviewer: APROVADO, sem BUG. Sonda HTTP no binário real e testes -race ok; sem redirecionamento aberto; login com erro uniforme. Riscos baixos: vazamento de existência de IDs (403 x 404) preexistente, slug login reservado, contraste Pearl.

---

author: ebrgr
created: 2026-11-23T01:42:00Z

---

Verificação: Reviewer APROVADO; sonda HTTP nos papéis; make check 13/13

---

author: evertonberger
created: 2026-11-23T01:42:00Z

---

Aceite do usuário em 22/11/2026 ('aceito a 128 e publique depois faça o push').

---

author: ebrgr
created: 2026-11-23T01:45:00Z

---

Publicação: versão 0.68 (tag `v0.68`), branch `feat/issue-128-remodelar-login-revisoes-e-mensagens` integrada ao `main` por fast-forward.

---
<!-- COMMENTS:END -->
