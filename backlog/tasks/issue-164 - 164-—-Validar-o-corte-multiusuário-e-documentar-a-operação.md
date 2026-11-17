---
id: ISSUE-164
title: 164 — Validar o corte multiusuário e documentar a operação
status: Aberta
assignee: []
created_date: '2026-11-17T17:33:00Z'
updated_date: '2026-11-17T17:33:00Z'
labels:
  - validacao
  - docs
  - operacao
milestone: m-10
dependencies:
  - ISSUE-158
  - ISSUE-159
  - ISSUE-160
  - ISSUE-161
  - ISSUE-162
  - ISSUE-163
references:
  - backlog/decisions/decision-010
documentation:
  - doc-041
type: chore
ordinal: 177000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: ISSUE-100 (aceita em 2026-11-17). Contrato normativo: doc-041 e decision-010. Padrão de senhas atual mantido (bcrypt e validação existentes), sem novos limites ou bloqueios por usuário ou origem; o limitador de login compartilhado existente fica como está; perfis existentes e novos nascem públicos (require_login=0); o relay segue ativo mesmo com escutas protegidas (consequência aceita). Escopo: validar o corte multiusuário inteiro e documentar a operação: matriz HTML, JSON e serviços, banco migrado sem perda, restauração e regressão de ingestão e relay, make check, evidências sem segredos, documentação do usuário e CHANGELOG.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Matriz de permissões e visibilidade validada de ponta a ponta; banco migrado sem perda e restauração testada
- [ ] #2 Regressão de ingestão e relay; make check verde; evidências sem segredos
- [ ] #3 Documentação operacional e CHANGELOG atualizados
<!-- AC:END -->
