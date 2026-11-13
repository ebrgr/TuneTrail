---
id: ISSUE-060
title: 060 — Adaptar issue-first ao Backlog.md e automatizar publicação com versão
status: Done
assignee: []
created_date: '2026-11-13T23:14:00Z'
updated_date: '2026-11-13T23:28:00Z'
labels:
  - processo
milestone: m-6
dependencies: []
references:
  - scripts/issue.py
  - scripts/test_issue.py
type: chore
ordinal: 60000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Adaptar a skill /.agents/issue-first ao fluxo do Backlog.md (sem GitHub Issues) e criar scripts make para iniciar, enviar para aceite e publicar issues aceitas: merge local fast-forward, incremento de versão por issue aceita (make version.bump), seção de versão no CHANGELOG, tag local e push opcional com PUSH=1. Commits levam o prefixo ISSUE-NNN: no título.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Skill issue-first descreve o ciclo do Backlog.md em vez do GitHub
- [ ] #2 make issue.start cria branch <tipo>/issue-NNN-slug e marca Em andamento
- [x] #3 make issue.publish só publica tarefa Done, com make check verde, versão incrementada, CHANGELOG e tag
- [x] #4 Push somente com PUSH=1
- [x] #5 Testes do script em repositório descartável
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-13T23:18:00Z

---

Verificação: skill movida para /.agents/issue-first (sem espaço) e reescrita para o ciclo do Backlog.md. Novo scripts/issue.py com make issue.new/start/review/publish/test. Publicação: exige Done + comentário 'Aceite', árvore limpa (exceto o registro da própria tarefa, que é commitado como 'ISSUE-NNN: Registra aceite'), commits com prefixo 'ISSUE-NNN: ', tag inexistente; faz rebase no main, make check, version.py --bump (0.x → 0.x+1), CHANGELOG [Unreleased] → [0.x] - data, commit 'ISSUE-NNN: Publica versão 0.x', --check-parent, merge --ff-only, tag anotada v0.x, remove a branch local; push só com PUSH=1. make issue.test: 15 testes OK (6 de version.py, 9 do fluxo em repositórios descartáveis com backlog falso). Dry-run real recusou ISSUE-060 por não estar Done. AC 2 (issue.start) não exercitado no repositório real: a árvore atual tem as próprias mudanças desta tarefa. agents.md e scripts/README.md atualizados; sem entrada no CHANGELOG (ferramenta interna).

---

author: claude
created: 2026-11-13T23:24:00Z

---

Refinamento solicitado pelo usuário: skill .agents/changelog reescrita de forma genérica (Keep a Changelog, ID da tarefa no fim da entrada, seção de publicação alinhada a make issue.publish). scripts/issue.py lê o prefixo de task_prefix em backlog/config.yml (ISS hoje, ISSUE se migrado). Migração ISS→ISSUE testada em cópia descartável (Backlog.md 1.55.3 lê, cria ISSUE-061, doctor limpo) mas NÃO aplicada: aguarda decisão do usuário. make issue.test: 23 testes OK, incluindo as jornadas com prefixo issue.

---

author: claude
created: 2026-11-13T23:27:00Z

---

Migração ISS→ISSUE aplicada a pedido do usuário: task_prefix "issue" em backlog/config.yml; 59 tarefas renomeadas com git mv (iss-NNN → issue-NNN) mais a ISSUE-060; referências ISS-NNN/iss-NNN reescritas em backlog/ (tarefas, docs, decisões, evidências de texto), agents.md, internal/webui/README.md, comentário de internal/imagepipeline/corpus_test.go e no fluxo novo. Variável do make passou de ISS= para ISSUE=. Verificação: backlog doctor limpo; 60 tarefas listadas como ISSUE-NNN; 864 links relativos válidos, 0 quebrados; make issue.test 23 testes OK (prefixo issue e outro prefixo configurado); make issue.publish ISSUE=60 DRY=1 recusa corretamente por não estar Done. Comentário anterior que narra a migração mantido como histórico. Commits já existentes (ex.: 1d47ecf) continuam citando ISS-059; histórico do git não é reescrito.

---

author: claude
created: 2026-11-13T23:28:00Z

---

Aceite: aprovada pelo usuário em 13/11/2026, incluindo a migração ISS→ISSUE. Critério 2 (make issue.start no repositório real) segue sem marcar: coberto apenas pela geração do nome da branch nos testes.

---
<!-- COMMENTS:END -->
