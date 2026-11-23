---
id: ISSUE-202
title: 202 — Cor de fundo do avatar escolhida numa lista
status: Em andamento
assignee:
  - '@ebrgr'
created_date: '2026-11-23T17:40:00Z'
updated_date: '2026-11-23T17:42:00Z'
labels:
  - ui
  - admin
milestone: m-10
dependencies:
  - ISSUE-201
references:
  - backlog/evidencias/issue-127/ESTUDO.md
type: feature
ordinal: 215000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: pedido do usuário em 23/11/2026, durante o teste do avatar. No cartão Avatar da aba Perfil, além do ícone e da foto, uma lista suspensa para escolher a cor de fundo do avatar (por exemplo #000000). Onde fica salvo: no banco, por perfil, em uma nova coluna listening_profiles.avatar_color (TEXT, nula = cor padrão do tema, com CHECK de formato #rrggbb). Isso exige uma migração (schema 22). O conf.toml guarda só configurações do servidor inteiro e não recebe nada desta issue. A lista de cores é uma constante no código (uma paleta fixa, com o preto); o servidor só aceita um valor da lista, nunca um valor livre, para que nada arbitrário chegue ao CSS. A cor vale onde o avatar aparece (cabeçalho do perfil, diretório, página do perfil, lista de contas, navegação), atrás das iniciais e do ícone; a foto cobre o fundo. Pontos a decidir no estudo: conteúdo da paleta, contraste do ícone e das iniciais sobre cada cor nos temas Yuu e Pearl, e se o ícone troca de cor para manter o contraste. Impacto da migração: backup automático antes de migrar, atualizar DB_ALTERATIONS.md e as jornadas de migração do e2e que fixam a versão 21, e alinhar com a ISSUE-195 (consolidação das migrações). Depende da ISSUE-201, que faz o formulário do avatar gravar.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Lista suspensa de cores no cartão Avatar; a cor escolhida é gravada em listening_profiles.avatar_color e continua escolhida depois de salvar
- [ ] #2 Migração 022 aditiva (coluna nula com CHECK de formato), com backup automático, e as jornadas de migração e o DB_ALTERATIONS.md atualizados
- [ ] #3 O servidor só aceita uma cor da paleta fixa; qualquer outro valor é recusado sem chegar ao CSS
- [ ] #4 A cor aparece onde o avatar aparece, com iniciais e ícone legíveis nos temas Yuu e Pearl; sem cor escolhida, o visual atual não muda
- [ ] #5 Funciona sem JavaScript; textos em pt-BR e en; teste de regressão e make check verde
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-23T17:42:00Z

---

Decisões do usuário em 23/11/2026: (1) paleta fixa no código, sem hex livre; (2) o ícone e as iniciais trocam de cor sozinhos nas cores escuras, para manter o contraste.

---

author: ebrgr
created: 2026-11-23T17:42:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-202` na branch `feat/issue-202-cor-de-fundo-do-avatar` criada a partir de `main`.

---
<!-- COMMENTS:END -->
