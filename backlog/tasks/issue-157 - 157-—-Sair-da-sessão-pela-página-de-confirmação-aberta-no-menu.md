---
id: ISSUE-157
title: 157 — Sair da sessão pela página de confirmação aberta no menu (+)
status: Aberta
assignee: []
created_date: '2026-11-17T12:46:00Z'
updated_date: '2026-11-17T12:46:00Z'
labels:
  - admin
  - ui
milestone: m-10
dependencies:
  - ISSUE-156
type: feature
ordinal: 170000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: parecer do Advisor na ISSUE-155; confirmação de saída aprovada pelo usuário em 2026-11-17 (Sair… abre uma página e depois o botão Sair). Hoje não há Sair na navegação: o formulário só existe na página de administração. Item 'Sair…' (ícone LogOut, Lucide ISC, a adicionar ao catálogo local; a chave continua só para entrar) no grupo Administração do menu (+) quando houver sessão, como LINK para a nova página GET /admin/sign-out. O GET só confirma e nunca revoga (nada de logout por prefetch). A página autenticada emite CSRF e ticket próprios (h.newSubmission) e um botão 'Sair' faz POST em /admin/logout existente (ParseSessionForm, consumeSubmission, admin.Logout, limpa cookies, 303 /admin); 'Cancelar' volta a um destino interno validado (SafeReturn/allowlist) ou /admin. Texto: 'Encerrar esta sessão neste navegador'. Não alterar a assinatura de admin.Logout nem o contrato da API POST /logout; não usar GET nem a API como fallback; não colocar token de logout em todas as páginas públicas nem em HTML em cache. Arquivos: internal/adminui/routing.go (IsRoute, Handler.dispatch, Handler.signOutPage), template admin (ou assets/templates/admin/sign-out.html), traduções en e pt-BR, testes adminui. Casos: sessão expirada entre render e clique, ticket reutilizado ou limite de 4096, duas abas, sem JavaScript.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Sair… aparece no menu (+) só com sessão administrativa válida e abre uma página de confirmação; o GET não revoga a sessão
- [ ] #2 O botão Sair faz o POST protegido existente, revoga só a sessão atual, limpa os cookies e redireciona para /admin; Cancelar volta a um destino interno validado
- [ ] #3 Provas de CSRF, sessão expirada, ticket reutilizado e duas abas; funciona sem JavaScript; ícone LogOut adicionado ao catálogo local
- [ ] #4 Aceite visual do usuário
<!-- AC:END -->
