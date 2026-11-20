---
id: ISSUE-157
title: 157 — Sair da sessão pela página de confirmação aberta no menu (+)
status: Done
assignee:
  - '@ebrgr'
created_date: '2026-11-17T12:46:00Z'
updated_date: '2026-11-20T21:24:00Z'
labels:
  - admin
  - ui
milestone: m-10
dependencies:
  - ISSUE-156
  - ISSUE-163
references:
  - backlog/evidencias/issue-163/ESTUDO.md
  - backlog/evidencias/issue-157/VERIFICACAO.md
type: feature
ordinal: 170000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: parecer do Advisor na ISSUE-155; confirmação de saída aprovada pelo usuário em 2026-11-17 (Sair… abre uma página e depois o botão Sair). Hoje não há Sair na navegação: o formulário só existe na página de administração. Item 'Sair…' (ícone LogOut, Lucide ISC, a adicionar ao catálogo local; a chave continua só para entrar) no grupo Administração do menu (+) quando houver sessão, como LINK para a nova página GET /admin/sign-out. O GET só confirma e nunca revoga (nada de logout por prefetch). A página autenticada emite CSRF e ticket próprios (h.newSubmission) e um botão 'Sair' faz POST em /admin/logout existente (ParseSessionForm, consumeSubmission, admin.Logout, limpa cookies, 303 /admin); 'Cancelar' volta a um destino interno validado (SafeReturn/allowlist) ou /admin. Texto: 'Encerrar esta sessão neste navegador'. Não alterar a assinatura de admin.Logout nem o contrato da API POST /logout; não usar GET nem a API como fallback; não colocar token de logout em todas as páginas públicas nem em HTML em cache. Arquivos: internal/adminui/routing.go (IsRoute, Handler.dispatch, Handler.signOutPage), template admin (ou assets/templates/admin/sign-out.html), traduções en e pt-BR, testes adminui. Casos: sessão expirada entre render e clique, ticket reutilizado ou limite de 4096, duas abas, sem JavaScript.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Sair… aparece no menu (+) só com sessão administrativa válida e abre uma página de confirmação; o GET não revoga a sessão
- [x] #2 O botão Sair faz o POST protegido existente, revoga só a sessão atual, limpa os cookies e redireciona para o diretório público, que oferece Entrar (reescopo de 20/11/2026; antes: /admin); Cancelar volta a um destino interno validado, com Conta ou diretório como fallback
- [x] #3 Provas de CSRF, sessão expirada, ticket reutilizado e duas abas; funciona sem JavaScript; ícone LogOut adicionado ao catálogo local
- [x] #4 Aceite visual do usuário
- [x] #5 Sair… disponível a qualquer sessão válida (não só administrativa) e presente na Conta e na 403; Cancelar volta a destino validado
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-18T21:42:00Z

---

Verificar na 157 (origem: Reviewer da ISSUE-159, 2026-11-18): usuários comuns agora autenticam, mas /admin/logout era restrito a administrador (corrigido minimamente na 159 para aceitar qualquer sessão ativa com CSRF). Ao implementar o sair pela página de confirmação do menu (+), validar o fluxo com sessão de usuário comum e com cookie de usuário comum no mesmo navegador de um admin (o admin deve conseguir sair e ver o login), e que o CSRF do logout é o da própria sessão.

---

author: claude
created: 2026-11-20T18:33:00Z

---

Reescopo (decisão do usuário em 20/11/2026, estudo da 163): Sair… atende comum e root, no menu (+), na Conta e na página 403, com a confirmação compartilhada. Revisar a proposta antiga do grupo Administração e do retorno fixo a /admin: o retorno é um destino interno validado, com fallback para Conta ou diretório; depois de sair, o diretório oferece Entrar.

---

author: ebrgr
created: 2026-11-20T21:03:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-157` na branch `feat/issue-157-sair-da-sessao-pela-pagina` criada a partir de `main`.

---

author: codex
created: 2026-11-20T21:12:00Z

---

Verificação conforme o reescopo de 20/11: confirmação para comum e root, Cancelar volta a leitura autorizada ou à Conta, e o POST encerra apenas a sessão atual e responde 303 para o diretório, que oferece Entrar. Testes Go e make check passaram (13/13 jornadas). Chromium validou a jornada por teclado sem JavaScript e a resposta da segunda aba. Oito capturas em Yuu e Pearl estão nas evidências. AC4 segue para aceite visual do usuário e a implementação segue para o Reviewer.

---

author: codex
created: 2026-11-20T21:23:00Z

---

Correções do Reviewer verificadas: ícone oficial Lucide log-out, retorno de Cancelar nas páginas administrativas e no diretório quando permitido, e comentários em inglês. Cancelar recebe o foco inicial e Tab chega a Sair sem JavaScript. A matriz de métodos e o CHANGELOG foram atualizados. Referência e oito capturas regeneradas. Testes Go dos pacotes afetados e make check passaram (13/13 jornadas). AC4 segue pendente para aceite visual.

---

author: claude
created: 2026-11-20T21:23:00Z

---

Texto do AC2 ajustado: o destino após sair é o diretório público (que oferece Entrar), conforme o reescopo de 20/11/2026 e o comportamento verificado e revisado; o texto original dizia /admin.

---

author: claude
created: 2026-11-20T21:24:00Z

---

Aceite do usuário em 20/11/2026: AC4 (aceite visual da página de confirmação de saída, testada ao vivo em servidor isolado com contas fictícias, e capturas em Yuu e Pearl). Reviewer: 3 itens obrigatórios corrigidos pelo Codex (ícone LogOut oficial do Lucide, Cancelar preservando a página de origem, comentários em inglês) mais ressalvas (matriz de métodos, foco inicial em Cancelar, CHANGELOG). Observação: o texto do AC1 («só com sessão administrativa válida») é anterior ao reescopo de 20/11/2026 e vale para qualquer sessão válida, como o AC5 registra.

---
<!-- COMMENTS:END -->
