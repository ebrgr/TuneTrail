---
id: ISSUE-163
title: 163 — Adequar telas multiusuário e navegação por permissão
status: Aberta
assignee: []
created_date: '2026-11-17T17:33:00Z'
updated_date: '2026-11-18T21:42:00Z'
labels:
  - ui
  - admin
  - i18n
milestone: m-10
dependencies:
  - ISSUE-160
  - ISSUE-161
  - ISSUE-162
references:
  - backlog/decisions/decision-010
documentation:
  - doc-041
type: enhancement
ordinal: 176000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: ISSUE-100 (aceita em 2026-11-17). Contrato normativo: doc-041 e decision-010. Padrão de senhas atual mantido (bcrypt e validação existentes), sem novos limites ou bloqueios por usuário ou origem; o limitador de login compartilhado existente fica como está; perfis existentes e novos nascem públicos (require_login=0); o relay segue ativo mesmo com escutas protegidas (consequência aceita). Escopo: telas de login e conta para o usuário comum, gestão própria e gestão do root distintas, checkbox de proteção com aviso de que o relay segue ativo, logout protegido do usuário comum (ver ISSUE-157), navegação por permissão. Preferir reescopar as ISSUE-125 a 128 e 157 em vez de duplicá-las; coordenar com 153 e 074.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Telas de login, conta, gestão própria e gestão root conforme o papel; aviso do relay junto à opção de proteção
- [ ] #2 Funciona sem JavaScript; foco, teclado e leitor de tela verificados
- [ ] #3 Capturas responsivas e aceite visual do usuário
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-18T21:42:00Z

---

Verificar na 163 (origem: Reviewer da ISSUE-159, 2026-11-18): (1) o login HTML de usuário comum hoje leva a /admin, que responde 403; a 159 só garante um logout mínimo na página 403 para não haver beco sem saída. Aqui deve entrar o destino correto do usuário comum após o login (sem página administrativa), a navegação por permissão e a página 403 definitiva com logout. (2) Confirmar que nenhuma tela ou link mostra ações administrativas a quem não é admin e que os logs e mensagens 401/403 ('Administrative login failed', 'administrator session required') ficaram corretos para contas comuns. (3) Reconferir o oráculo de tempo do login após a correção da 159 (hash fixo nos ramos conta inexistente e pendente).

---
<!-- COMMENTS:END -->
