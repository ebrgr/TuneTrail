---
id: ISSUE-116
title: >-
  116 — Estudo: identidade de faixas entre lançamentos, títulos iguais, aliases
  e caixa
status: Done
assignee:
  - '@ebrgr'
created_date: '2026-11-14T22:54:00Z'
updated_date: '2026-11-15T16:44:00Z'
labels:
  - catalogo
milestone: m-13
dependencies: []
documentation:
  - doc-036
type: spike
ordinal: 125000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: estudo ISSUE-114 (doc-034). Investigar a identidade de faixas: uma faixa em vários lançamentos (Koito #34, #217, roadmap Koito 0.4), títulos iguais no mesmo álbum (PR #315), importação Last.fm que junta faixas homônimas de álbuns diferentes (#214), alias secundário ignorado na associação por título (#369; confirmado: trackByTrackInfo só consulta tracks_with_title), normalização de caixa (#359) e fusão opcional de remaster/deluxe (#360). Relacionar com ISSUE-081 (deduplicação) e ISSUE-096 (grupos sem fundir). Saída: proposta de modelo e migração, sem implementação.
<!-- SECTION:DESCRIPTION:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: ebrgr
created: 2026-11-15T15:48:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-116` na branch `spike/issue-116-estudo-identidade-de-faixas-entre` criada a partir de `chore/issue-129-integracao-com-o-orca-worktrees`.

---

author: codex
created: 2026-11-15T15:54:00Z

---

Estudo doc-036 concluído: reproduções em SQLite em memória confirmaram perda de lançamento com MBID, separação sem MBID e recriação após fusão, colisão de títulos/MBIDs, alias secundário ignorado e diferenças de caixa; Last.fm sem MBID preserva álbuns e com MBID compartilhado junta. Recomenda separar gravação/ocorrência e grupos reversíveis por perfil; três perguntas registradas. Teste temporário removido; nenhum arquivo de produção alterado.

---

author: ebrgr
created: 2026-11-15T15:54:00Z

---

Verificação: estudo doc-036; reproduções fictícias de identidade PASS; sem alterações de produção; make check aprovado

---

author: codex
created: 2026-11-15T15:54:00Z

---

Verificação final: make issue.review ISSUE=116 executado com sucesso; make check aprovado (vet, testes Go, UI/JS/Python/estilo e 12/12 jornadas E2E). Estudo doc-036 revisado; somente documentação e este registro alterados, teste temporário removido. ISSUE-116 em Aguardando aceite do usuário; sem aceite, publicação, push ou merge.

---

author: claude
created: 2026-11-15T16:42:00Z

---

Decisões do usuário registradas no doc-036: (1) ranking de faixas com chave na interface 'por gravação' / 'por lançamento', padrão por gravação; (2) maiúsculas e minúsculas associadas automaticamente quando há uma única candidata e nenhum conflito de MBID, preservando a grafia original, com registro e desfazer; (3) remaster/deluxe com agrupamento por perfil e também fusão global com prévia na administração, escolhidos por configuração (novo recorte 8).

---

author: claude
created: 2026-11-15T16:42:00Z

---

Aceite do usuário em 15/11/2026. Integração por fast-forward na main, sem versão (estudo sem mudança no produto). Primeiro estudo feito por um worker Codex numa worktree do Orca (piloto da ISSUE-129).

---

author: claude
created: 2026-11-15T16:44:00Z

---

Recortes registrados: ISSUE-131 a ISSUE-138, na ordem do doc-036 (131–134 e 136 na Fase 2, 135, 137 e 138 na Fase 4).

---
<!-- COMMENTS:END -->
