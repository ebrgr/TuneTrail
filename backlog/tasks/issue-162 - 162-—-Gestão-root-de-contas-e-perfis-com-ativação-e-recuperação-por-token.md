---
id: ISSUE-162
title: 162 — Gestão root de contas e perfis com ativação e recuperação por token
status: Em andamento
assignee:
  - '@ebrgr'
created_date: '2026-11-17T17:33:00Z'
updated_date: '2026-11-19T17:32:00Z'
labels:
  - admin
  - banco
  - api
milestone: m-10
dependencies:
  - ISSUE-158
  - ISSUE-159
  - ISSUE-160
references:
  - backlog/decisions/decision-010
  - backlog/evidencias/issue-162/VERIFICACAO.md
documentation:
  - doc-041
priority: high
type: feature
ordinal: 175000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: ISSUE-100 (aceita em 2026-11-17). Contrato normativo: doc-041 e decision-010. Padrão de senhas atual mantido (bcrypt e validação existentes), sem novos limites ou bloqueios por usuário ou origem; o limitador de login compartilhado existente fica como está; perfis existentes e novos nascem públicos (require_login=0); o relay segue ativo mesmo com escutas protegidas (consequência aceita). Escopo: somente o root cadastra e exclui contas e perfis (operação conjunta, atômica e confirmada, que remove sessões, tokens, chaves e relay do perfil sem tocar nos outros); qualquer perfil pode ser promovido a admin; ativação sem senha compartilhada e recuperação sem e-mail por token temporário emitido pelo root (uso único, expira, reset revoga sessões); o último admin não pode ser excluído nem rebaixado.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Cadastro e exclusão só pelo root e atômicos; promoção a admin por outro admin; último admin protegido
- [x] #2 Token temporário de uso único, com expiração; reset revoga sessões; testes de token expirado e reutilizado
- [x] #3 Nas superfícies web e API da ISSUE-162, o root nunca vê senha nem hash em tela, resposta ou log.
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: ebrgr
created: 2026-11-19T16:34:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-162` na branch `feat/issue-162-gestao-root-de-contas-e` criada a partir de `main`.

---

author: codex
created: 2026-11-19T16:35:00Z

---

Verificação inicial: migração 007 livre na main v0.40. Implementação pausada conforme instrução da tarefa até confirmação de validade dos tokens, emissão para admin e autoexclusão. Evidência vinculada; nenhum AC marcado.

---

author: ebrgr
created: 2026-11-19T16:59:00Z

---

Decisões do usuário em 19/11/2026: ativação em 24 horas e recuperação em 30 minutos, por constantes. Root pode emitir para outra conta, inclusive outro admin, e é vedado emitir para si mesmo. O único admin recupera acesso localmente, sem token. Autoexclusão confirmada é permitida quando houver outro admin ativo; o último continua protegido.

---

author: codex
created: 2026-11-19T16:59:00Z

---

Verificação de contrato: decisões de ebrgr registradas no doc-041. Novo conflito: reset-password local publica a nova senha pelo logger estruturado (engine/reset_password.go:46). Implementação pausada para decidir se o AC #3 inclui corrigir a entrega da senha no comando local. Evidência atualizada; nenhum AC marcado.

---

author: ebrgr
created: 2026-11-19T17:17:00Z

---

Decisão de 19/11/2026: o reset-password local segue gerando a senha, revogando sessões e publicando a nova senha no log do processo. O AC #3 cobre as superfícies web e API da ISSUE-162. O operador aceita o risco do comando local e deve proteger os logs.

---

author: codex
created: 2026-11-19T17:32:00Z

---

Verificação: make check passou com 13/13 jornadas E2E e go test -race passou nos seis pacotes verificados, inclusive disputa em WAL. Os três critérios foram comprovados. Inventário de rotas, ciclo do token e evidências em backlog/evidencias/issue-162/VERIFICACAO.md. Tarefa segue Em andamento.

---
<!-- COMMENTS:END -->
