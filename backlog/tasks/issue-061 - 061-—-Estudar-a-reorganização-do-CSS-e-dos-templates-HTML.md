---
id: ISSUE-061
title: 061 — Estudar a reorganização do CSS e dos templates HTML
status: Aguardando aceite
assignee: []
created_date: '2026-11-13T23:47:00Z'
updated_date: '2026-11-13T23:47:00Z'
labels:
  - ui
milestone: m-8
dependencies: []
documentation:
  - doc-031
type: spike
ordinal: 61000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Estudo para deixar o CSS e os templates HTML legíveis e fáceis de manter, sem Tailwind (ISSUE-058, decision-006) e sem etapa de build. Responde se os arquivos são compilados no binário e propõe movê-los para templates/ e assets/ na raiz. Estudo: doc-031.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Responder se CSS e templates são compilados no binário ou lidos do disco
- [x] #2 Medir a situação atual (tamanhos, linhas, pontos de quebra, dependências de caminho)
- [x] #3 Comparar localizações (internal, raiz embutido, raiz externo, híbrido) e recomendar
- [x] #4 Propor organização do CSS e dos templates com método de verificação por fase
- [x] #5 Dividir a implementação em fases que viram tarefas separadas
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-13T23:47:00Z

---

Verificação: estudo doc-031 escrito a partir de leitura de internal/webui/preview.go:21, internal/adminui/admin.go:35, internal/webui/assets.go, internal/i18n/i18n.go (precedente langs/), Dockerfile e medição de todos os templates e CSS (tamanho, linhas, maior linha, media queries). Sem mudança de código. Aguarda decisões do usuário: embutido+dev (recomendado) ou externo; nomes templates/ e assets/; fonte local na mesma leva ou separada.

---
<!-- COMMENTS:END -->
