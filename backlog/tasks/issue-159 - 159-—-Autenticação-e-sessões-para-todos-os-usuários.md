---
id: ISSUE-159
title: 159 — Autenticação e sessões para todos os usuários
status: Em andamento
assignee:
  - '@ebrgr'
created_date: '2026-11-17T17:33:00Z'
updated_date: '2026-11-18T21:47:00Z'
labels:
  - admin
  - api
milestone: m-10
dependencies:
  - ISSUE-158
references:
  - backlog/decisions/decision-010
  - backlog/evidencias/issue-159/VERIFICACAO.md
  - backlog/evidencias/issue-159/PARECER-REVIEWER.md
documentation:
  - doc-041
priority: high
type: feature
ordinal: 172000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: ISSUE-100 (aceita em 2026-11-17). Contrato normativo: doc-041 e decision-010. Padrão de senhas atual mantido (bcrypt e validação existentes), sem novos limites ou bloqueios por usuário ou origem; o limitador de login compartilhado existente fica como está; perfis existentes e novos nascem públicos (require_login=0); o relay segue ativo mesmo com escutas protegidas (consequência aceita). Escopo: o login passa a valer para usuários user e admin (hoje só admin autentica), separando autenticar de autorizar; separar sessão de papel; CSRF e tickets adaptados; troca de senha revoga as sessões e a corrida com login de senha antiga é coberta; sem novos limites ou bloqueios.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Login user e admin com bcrypt e validação atuais; sessão por usuário, isolada
- [x] #2 CSRF e tickets adaptados ao usuário comum; troca de senha revoga sessões; corrida com senha antiga coberta
- [x] #3 Nenhum limite novo por usuário ou origem; limitador compartilhado existente intacto (teste)
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: ebrgr
created: 2026-11-18T21:25:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-159` na branch `feat/issue-159-autenticacao-e-sessoes-para-todos` criada a partir de `main`.

---

author: codex
created: 2026-11-18T21:30:00Z

---

Verificação Codex: autenticação active para user/admin separada de autorização; AdminIdentity preserva todas as fronteiras HTML administrativas. CSRF comum e tickets isolados; revogação por conta e corrida com senha antiga testadas; limitador compartilhado original intacto e testado. make check passou (13/13 e2e e referência visual); go test -race passou em admin, adminweb, adminui, sqlite, webui e engine. Evidência: backlog/evidencias/issue-159/VERIFICACAO.md. ACs 1–3 comprovados; gestão própria e telas permanecem nas 160–163. Mantida Em andamento para revisão do responsável pelo despacho; sem push/merge/publish/Done.

---

author: reviewer
created: 2026-11-18T21:36:00Z

---

Parecer Reviewer: APROVADO COM RESSALVAS. Sem escalonamento: todas as rotas JSON/HTML administrativas exigem role admin (RequireAdmin/AdminIdentity), pending recusado em todos os caminhos, CSRF/tickets ligados à sessão, corrida login×troca de senha coberta, limitador global intacto; go test -race (admin, adminweb, adminui, sqlite, webui, engine) passou. Ressalvas: (1) login HTML de user leva a /admin 403 sem logout HTML e bloqueia o formulário de login de admin no mesmo navegador até expirar — corrigir ou registrar como gate da 163/157; (2) oráculo de tempo no Login (sem bcrypt para inexistente/pending) agora cobre todas as contas — sugerido hash fictício; (3) lacunas de teste: /{slug}/manage, webui.adminSession e /admin/logout com sessão user; (4) mensagens/log 'Administrative'. Advisor recomendado (autenticação).

---

author: codex
created: 2026-11-18T21:47:00Z

---

Correções solicitadas dos achados 1–3 do Reviewer executadas diretamente, sem re-delegação: /admin 403 de user agora oferece mensagem en/pt-BR e logout HTML com CSRF/ticket da própria sessão; apenas POST /admin/logout aceita SessionIdentity, demais rotas administrativas seguem restritas. Login executa bcrypt DefaultCost fictício para inexistente/pending (teste por função interna injetável, sem tempo); log e 401 genérico ajustados. Regressões adicionadas para GET/POST de mídia, adminSession, HTML público e login/logout user/admin. make check passou (13/13 e2e); go test -race passou nos seis pacotes alterados. Referências UI atualizadas somente em data-ui-generation, com demais bytes verificados idênticos. Evidência em VERIFICACAO.md; parecer/comentário original preservados e incluídos no commit. ACs mantidos; status Em andamento. Sem push/merge/publish/Done.

---
<!-- COMMENTS:END -->
