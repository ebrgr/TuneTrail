---
id: ISSUE-150
title: 150 — Administrador vê a página vazia de faixa, álbum ou artista sem escutas
status: Done
assignee:
  - '@ebrgr'
created_date: '2026-11-16T14:57:00Z'
updated_date: '2026-11-17T15:23:00Z'
labels:
  - ui
  - admin
milestone: m-10
dependencies: []
type: bug
ordinal: 163000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: ISSUE-144 item 3 (parecer da ISSUE-141). Decisão do usuário em 16/11/2026: depois de apagar a última escuta, o administrador vê a página com estado vazio também em GET normal (hoje o diálogo mostra o estado vazio, mas recarregar dá 404). Visitante e outros perfis continuam com 404.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 O administrador vê a página vazia de faixa, álbum ou artista sem escutas em GET normal, também ao recarregar
- [x] #2 Visitantes e entidades inexistentes continuam com 404
- [x] #3 Testes cobrem recarga do administrador, visitante e entidade inexistente; make check verde
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: ebrgr
created: 2026-11-17T12:57:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-150` na branch `fix/issue-150-administrador-ve-a-pagina-vazia` criada a partir de `main`.

---

author: claude
created: 2026-11-17T12:59:00Z

---

Verificação: AllowEmptyMedia passou a valer para qualquer GET do administrador (antes exigia o cabeçalho X-TuneTrail-History-Refresh). Testes: engine/adminui_listen_141_test (recarga 200 com estado vazio, visitante 404, entidade inexistente 404 para o admin) e adminui_tasks_139_test (sobrevivente sem escutas: admin 200, visitante 404). make check verde (12/12 e2e). O cabeçalho no media-dialog.js ficou inofensivo e redundante; remoção opcional.

---

author: ebrgr
created: 2026-11-17T15:23:00Z

---

Aceite do usuário (2026-11-17). Administrador vê o estado vazio de faixa, álbum e artista sem escutas também ao recarregar; visitante e entidade inexistente seguem com 404. Publicação local; push depende de autorização à parte.

---
<!-- COMMENTS:END -->
