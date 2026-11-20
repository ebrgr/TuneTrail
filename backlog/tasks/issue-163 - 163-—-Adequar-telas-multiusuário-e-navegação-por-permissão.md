---
id: ISSUE-163
title: 163 — Adequar telas multiusuário e navegação por permissão
status: Em andamento
assignee:
  - '@ebrgr'
created_date: '2026-11-17T17:33:00Z'
updated_date: '2026-11-20T20:27:00Z'
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
  - backlog/evidencias/issue-163/ESTUDO.md
  - backlog/evidencias/issue-163/VERIFICACAO.md
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
- [x] #1 Telas de login, conta, gestão própria e gestão root conforme o papel; aviso do relay junto à opção de proteção
- [ ] #2 Funciona sem JavaScript; foco, teclado e leitor de tela verificados
- [ ] #3 Capturas responsivas e aceite visual do usuário
- [x] #4 Modelo de navegação montado no servidor a partir de AuthorizeAction: o HTML de visitante, comum, dono e root não traz links ou ações que o papel não pode usar; chamadas diretas continuam negadas
- [x] #5 Login válido segue para /account (comum) ou /admin (root), com retorno prioritário a leitura autorizada e validada; 403 definitivo oferece Minha conta, Diretório e Sair…
- [x] #6 Textos e logs sem «administrator session required» e «Administrative logout failed» para o comum; 404 legado com Vary: Cookie e política de cache; cadeado acessível; teste de tempo do login reexecutado
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-18T21:42:00Z

---

Verificar na 163 (origem: Reviewer da ISSUE-159, 2026-11-18): (1) o login HTML de usuário comum hoje leva a /admin, que responde 403; a 159 só garante um logout mínimo na página 403 para não haver beco sem saída. Aqui deve entrar o destino correto do usuário comum após o login (sem página administrativa), a navegação por permissão e a página 403 definitiva com logout. (2) Confirmar que nenhuma tela ou link mostra ações administrativas a quem não é admin e que os logs e mensagens 401/403 ('Administrative login failed', 'administrator session required') ficaram corretos para contas comuns. (3) Reconferir o oráculo de tempo do login após a correção da 159 (hash fixo nos ramos conta inexistente e pendente).

---

author: ebrgr
created: 2026-11-19T20:00:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-163` na branch `feat/issue-163-adequar-telas-multiusuario-e-navegacao` criada a partir de `main`.

---

author: codex
created: 2026-11-19T20:04:00Z

---

Estudo documental concluído na base v0.43. Registra inventário por papel, caminhos para 403, débitos das ISSUE-159 a 162, proposta de reescopo da 125–128 e 157, matriz de verificação futura e decisões para aceite do desenho. ACs seguem abertos.

---

author: claude
created: 2026-11-20T18:33:00Z

---

Decisões do usuário em 20/11/2026 sobre o estudo (backlog/evidencias/issue-163/ESTUDO.md), as 4 aprovadas: (1) destino pós-login: comum em /account, root em /admin, com retorno prioritário a leitura autorizada; (2) entradas neutras /login, /account e «Meu perfil», com compatibilidade das rotas atuais; (3) as telas saem desta tarefa e vão para 125–128 e 157, e a 163 fica com integração, permissões, retorno do login e aceite (débitos de lock do resgate inválido e de CreateListeningProfile/DeleteListeningProfile vão para tarefas técnicas próprias; mudança de validade do token ao trocar papel exige decisão separada); (4) «Sair…» no menu (+), na Conta e na 403, com confirmação compartilhada. Ordem: 163 (modelo de navegação e retorno) → 157 e 125 → 126 e 128 → 164.

---

author: ebrgr
created: 2026-11-20T20:06:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-163` na branch `feat/issue-163-adequar-telas-multiusuario-e-navegacao` criada a partir de `main`.

---

author: codex
created: 2026-11-20T20:27:00Z

---

Verificação da implementação. Navegação e ações renderizadas por permissão, retorno pós-login validado, Conta na rota funcional até a 125, saída protegida até a 157, 403/404 e cache verificados, aviso do relay e pendências da 161 cobertos. Testes Go, instrumentação de login, custo das fixtures migradas e make check passaram (13/13 jornadas E2E). Foram registradas 56 capturas em Yuu/Pearl e a jornada sem JavaScript com teclado e árvore de acessibilidade. O AC 2 segue aberto para validação com leitor de tela em execução e o AC 3 para o aceite visual do usuário. A tarefa permanece Em andamento para o Reviewer.

---
<!-- COMMENTS:END -->
