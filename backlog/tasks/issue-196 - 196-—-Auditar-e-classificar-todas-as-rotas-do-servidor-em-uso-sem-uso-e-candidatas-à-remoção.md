---
id: ISSUE-196
title: >-
  196 — Auditar e classificar todas as rotas do servidor: em uso, sem uso e
  candidatas à remoção
status: Done
assignee:
  - '@ebrgr'
created_date: '2026-11-23T00:58:00Z'
updated_date: '2026-11-23T01:13:00Z'
labels:
  - api
milestone: m-12
dependencies:
  - ISSUE-194
references:
  - backlog/evidencias/issue-196/ROTAS.md
  - backlog/evidencias/issue-196/REMOCOES.md
  - backlog/evidencias/issue-196/OPENAPI.md
type: chore
ordinal: 209000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: pedido do usuário em 22/11/2026. Levantar TODAS as rotas HTTP do servidor (engine/routes.go, engine/handlers, internal/adminui e internal/webui, internal/adminweb, rotas de imagens, estáticos e a API ListenBrainz em /apis/listenbrainz/1) e classificar cada uma como EM USO (interface web, JS, scripts E2E, clientes ListenBrainz como Navidrome e Pano, importadores, exportação, relay), SEM USO (nenhum chamador na interface, nos scripts, nos testes de produto nem em cliente conhecido), LEGADA/COMPATIBILIDADE (rotas herdadas do Koito mantidas de propósito) ou A DECIDIR. Para cada rota registrar: método e caminho, handler, autenticação e autorização exigidas (visitante, sessão comum, dono, root, chave de API), quem chama (arquivo ou cliente), cobertura de teste, estabilidade pública (contrato externo ou interno) e recomendação (manter, documentar, restringir, marcar como obsoleta ou remover). Entregar tabela em backlog/evidencias/issue-196/ROTAS.md e a lista de remoções propostas, cada uma com o risco. Método: extrair as rotas do código (chi e dispatchers fechados IsPage/IsAction), cruzar com grep nos templates, assets/js e scripts, e com os logs de acesso de um ensaio; usar graphify para os chamadores. A classificação alimenta a ISSUE-104 (OpenAPI e métricas Prometheus), que virá depois: só rotas EM USO e estáveis entram na especificação, e as de sessão e administração ficam marcadas como internas; ao fechar a 196, registrar na 104 quais rotas documentar, quais ficam fora e quais mudam antes. Relaciona-se com a ISSUE-149 (código morto da exclusão global do catálogo), a 121 (escopos de chaves de API e rotas só de sessão), a 145 (contrato para apps nativos) e a 194 (código legado). Remoções ficam em issues próprias depois da aprovação da lista; esta issue não apaga nada. Critérios: toda rota catalogada; nenhuma remoção sem chamador comprovadamente ausente e teste; rotas de contrato externo (ListenBrainz, imagens, exportação) nunca marcadas como sem uso sem evidência. Reviewer ao final.
<!-- SECTION:DESCRIPTION:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-23T01:00:00Z

---

Início (22/11/2026): despachada ao Codex em worktree própria, a pedido do usuário. Somente leitura e documentação: nada é removido.

---

author: ebrgr
created: 2026-11-23T01:00:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-196` na branch `chore/issue-196-auditar-e-classificar-todas-as` criada a partir de `main`.

---

author: codex
created: 2026-11-23T01:08:00Z

---

Auditoria concluída nesta worktree: 225 padrões da produção (145 em uso, 39 de compatibilidade, 9 combinações inválidas sem uso e 32 a decidir), mais cinco da prévia. A extração reproduz o inventário. Três percursos E2E passaram em 4130 com SQLite descartável e log sanitizado. TestRouteManifest, TestAssetHTTPContracts e TestExternalAssetExposure passaram. As propostas de restrição e o recorte para a ISSUE-104 estão nas evidências. A tarefa permanece Em andamento para revisão e aceite.

---

author: claude
created: 2026-11-23T01:09:00Z

---

Reviewer: APROVADO, sem BUG; extrator reproduz 225 padrões; diff de produção vazio; 9 candidatas a restringir e 32 a decidir; insumo para a 104 em OPENAPI.md.

---

author: ebrgr
created: 2026-11-23T01:13:00Z

---

Verificação: Reviewer APROVADO; auditoria somente leitura, diff de produção vazio; extrator reproduz 225 padrões

---

author: evertonberger
created: 2026-11-23T01:13:00Z

---

Aceite do usuário em 22/11/2026 ('aceito a 196 após publique').

---
<!-- COMMENTS:END -->
