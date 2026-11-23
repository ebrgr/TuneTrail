---
id: ISSUE-164
title: 164 — Validar o corte multiusuário e documentar a operação
status: Done
assignee:
  - '@ebrgr'
created_date: '2026-11-17T17:33:00Z'
updated_date: '2026-11-23T19:20:00Z'
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
  - ISSUE-163
references:
  - backlog/decisions/decision-010
  - backlog/evidencias/issue-163/ESTUDO.md
  - backlog/evidencias/issue-164/MATRIZ.md
  - backlog/evidencias/issue-164/MIGRACAO.md
  - backlog/evidencias/issue-164/RESTAURACAO.md
  - backlog/evidencias/issue-164/REGRESSAO.md
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
- [x] #1 Matriz de permissões e visibilidade validada de ponta a ponta; banco migrado sem perda e restauração testada
- [x] #2 Regressão de ingestão e relay; make check verde; evidências sem segredos
- [x] #3 Documentação operacional e CHANGELOG atualizados
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-20T18:33:00Z

---

Estudo da 163 (20/11/2026): recebe a matriz de papéis e estados (visitante, comum, dono, comum em perfil de terceiro, root, sessão expirada ou rebaixada), a validação ampla e a documentação da operação. Depende da 163.

---

author: ebrgr
created: 2026-11-23T19:00:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-164` na branch `chore/issue-164-validar-o-corte-multiusuario-e` criada a partir de `main`.

---

author: codex
created: 2026-11-23T19:10:00Z

---

Verificação da v0.72 e do schema 22. Matriz HTML, JSON e serviços conferida com sessões de visitante, dono, terceiro, root, rebaixada e expirada. A jornada Chrome passou com JavaScript ligado e desligado. A cópia real v2 e a restauração preservaram 72 escutas e o mesmo SHA-256. O backup real v4 preservou 80 escutas na migração e na restauração; v3 foi exercitado com fixture. Ingestão e relay passaram. make check verde com 13/13 jornadas e zero diferenças em 190 arquivos de referência; build dos docs verde. Guia e changelog atualizados. A ISSUE-197 já acompanha a configuração LOGIN_GATE. A tarefa segue em andamento para o parecer do Reviewer pelo agente que despachou.

---

author: claude
created: 2026-11-23T19:18:00Z

---

Aceite: aprovada pelo usuário em 23/11/2026. Parecer do Reviewer: APROVADO (backlog/evidencias/issue-164/PARECER-REVIEWER.md). Verificado: matriz de papéis e estados em HTML, JSON e serviços, com JavaScript ligado e desligado; migração do schema 2 ao 22 sem perda (72 escutas e mesmo digest antes, depois e após restaurar o backup); regressão de ingestão e relay; make check verde com 13/13 jornadas; guia operacional e CHANGELOG. Riscos conhecidos: as menções a schema 22 nos guias e no DEPLOYMENT.md precisam ser atualizadas ao integrar a ISSUE-190 (schema 23); o teste de navegador exige Playwright e Chromium e não está no make check; inspeção de teclado e leitor de tela fora desta execução. Publicada localmente, sem push.

---

author: ebrgr
created: 2026-11-23T19:20:00Z

---

Publicação: versão 0.73 (tag `v0.73`), branch `chore/issue-164-validar-o-corte-multiusuario-e` integrada ao `main` por fast-forward.

---
<!-- COMMENTS:END -->
