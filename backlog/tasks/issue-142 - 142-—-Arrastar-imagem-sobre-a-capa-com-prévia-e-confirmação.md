---
id: ISSUE-142
title: 142 — Arrastar imagem sobre a capa com prévia e confirmação
status: Aberta
assignee: []
created_date: '2026-11-15T20:04:00Z'
labels:
  - ui
  - imagens
milestone: m-10
dependencies:
  - ISSUE-139
  - ISSUE-140
references:
  - backlog/evidencias/issue-108/avaliacao-codex.md
priority: medium
type: feature
ordinal: 155000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Recorte D da ISSUE-108 (avaliação do Codex em backlog/evidencias/issue-108/avaliacao-codex.md, §5). Administrador arrasta uma imagem sobre a capa de álbum ou artista e vê prévia com confirmar e cancelar; a confirmação usa o mesmo formulário e o ReplaceArtwork do servidor, sem endpoint novo, e a capa é atualizada sem recarregar a página. Nada é enviado antes de confirmar. A página de imagem continua como alternativa por teclado e input de arquivo. Reaproveita só a função de arrasto do admin.js, sem trazer o comportamento global dele para a página pública.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Nada é enviado antes de confirmar; cancelar preserva a capa e libera a prévia
- [ ] #2 Arquivo inválido, grande, múltiplo, sessão expirada e falha de rede têm resposta clara, sem repetição automática
- [ ] #3 Ao concluir, a capa e as variantes de tamanho são atualizadas sem recarregar; foco preservado; alternativa por teclado funciona
- [ ] #4 Capturas em desktop e celular e aceite visual do usuário
<!-- AC:END -->
