---
id: ISSUE-166
title: >-
  166 — Títulos das páginas: remover o Koito da Home e padronizar com o nome do
  perfil e a tradução
status: Em andamento
assignee:
  - '@ebrgr'
created_date: '2026-11-17T20:48:00Z'
updated_date: '2026-11-28T02:04:00Z'
labels:
  - ui
  - i18n
milestone: m-10
dependencies: []
references:
  - backlog/evidencias/issue-166/reference-changes.txt
  - backlog/evidencias/issue-166/PARECER-REVIEWER.md
priority: low
type: bug
ordinal: 179000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: pergunta do usuário em 2026-11-17 (a aba da Home mostrava 'Koito - TuneTrail'). Causa: internal/webui/handlers.go:162 define p.Title = "Koito" para a Home, resto do fork original (commit 5c1affa); o sufixo ' - TuneTrail' vem do shared/head.html. Os demais títulos são 'Last Played', 'Top Artists', 'Search' e 'Your October 2026 Rewind' (em inglês, mesmo com a interface em português) e 'Administration — TuneTrail' na administração. Proposta: o título da Home passa a ser o nome do perfil (por exemplo 'Demo cheio - TuneTrail'), sem a palavra Koito; os títulos das demais páginas são traduzidos conforme o idioma (en e pt-BR) e seguem um padrão único '<página> - TuneTrail' (ou '<perfil> · <página> - TuneTrail', a decidir). Intencionais, que ficam: o rodapé 'baseado no Koito 0.3.2' e a opção 'Koito' no formato de importação. Nenhum outro 'Koito' visível foi achado no código das páginas.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 A Home não mostra mais 'Koito' no título da aba; usa o nome do perfil e o sufixo TuneTrail
- [x] #2 Os títulos de histórico, rankings, busca, retrospectiva e mídia seguem o idioma escolhido (en e pt-BR) e um padrão único com o sufixo TuneTrail
- [x] #3 Testes cobrem os títulos em en e pt-BR; o rodapé de atribuição e a opção de importação do Koito permanecem
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: ebrgr
created: 2026-11-28T01:49:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-166` na branch `fix/issue-166-titulos-das-paginas-remover-o` criada a partir de `main`.

---

author: claude
created: 2026-11-28T01:49:00Z

---

Decisão do usuário em 27/11/2026: a página inicial usa «<nome do perfil> - TuneTrail»; as demais usam «<página traduzida> - TuneTrail». O perfil não entra no título das outras páginas. Implementação despachada ao agente DeepSeek (dsh).

---

author: deepseek
created: 2026-11-28T02:00:00Z

---

Verificação (deepseek): títulos conforme a decisão — a Home usa o nome do perfil (p.Title = p.Profile.Name em internal/webui/handlers.go; o sufixo ' - TuneTrail' vem do head compartilhado) e as demais páginas usam o nome traduzido pela chave page.* (en.json e pt-BR.json); a palavra Koito saiu de todos os títulos. O rodapé de atribuição (Koito 0.3.2 com o logo) e a opção Koito no formato de importação permanecem. Testes novos em internal/webui/titles_issue166_test.go cobrem, em en e pt-BR: Home, diretório de perfis, histórico, os três rankings, busca, retrospectiva (anual e mensal), mídia e páginas de erro; asseguram a ausência de Koito no título e a permanência do rodapé e da opção de importação. Referência de UI regenerada (só public/home.html e public/empty.html mudam: o título). make check verde (e2e 13/13).

---

author: claude
created: 2026-11-28T02:04:00Z

---

Reviewer: APROVADO (backlog/evidencias/issue-166/PARECER-REVIEWER.md).

---
<!-- COMMENTS:END -->
