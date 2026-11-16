---
id: ISSUE-142
title: 142 — Arrastar imagem sobre a capa com prévia e confirmação
status: Done
assignee:
  - '@ebrgr'
created_date: '2026-11-15T20:04:00Z'
updated_date: '2026-11-16T14:49:00Z'
labels:
  - ui
  - imagens
milestone: m-10
dependencies:
  - ISSUE-139
  - ISSUE-140
references:
  - backlog/evidencias/issue-108/avaliacao-codex.md
  - backlog/evidencias/issue-142/reviewer.md
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
- [x] #1 Nada é enviado antes de confirmar; cancelar preserva a capa e libera a prévia
- [x] #2 Arquivo inválido, grande, múltiplo, sessão expirada e falha de rede têm resposta clara, sem repetição automática
- [x] #3 Ao concluir, a capa e as variantes de tamanho são atualizadas sem recarregar; foco preservado; alternativa por teclado funciona
- [ ] #4 Capturas em desktop e celular e aceite visual do usuário
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: ebrgr
created: 2026-11-16T14:31:00Z

---

Início: branch `feat/issue-142-arrastar-imagem-sobre-a-capa` criada a partir de `main`.

---

author: codex
created: 2026-11-16T14:36:00Z

---

Implementação: o drop agora é limitado à capa de mídia para administradores; abre a tarefa de imagem existente, injeta o arquivo no mesmo formulário sem POST, mostra prévia local, oferece confirmar/cancelar e revoga object URLs ao cancelar, trocar ou concluir. O refresh autoritativo do cabeçalho atualiza src/srcset sem recarregar a página. Validações de tipo, tamanho e quantidade dão erro claro sem retry. Verificações: node --check, git diff --check e testes de internal/adminui com TUNETRAIL_WEB_DIR passaram; make check não concluiu porque o sandbox bloqueia bind TCP em testes existentes; graphify AST também foi bloqueado ao criar cache externo. Critérios #1–#3 não marcados por falta de execução em navegador/integração; #4 permanece pendente para aceite visual.

---

author: claude
created: 2026-11-16T14:43:00Z

---

Parecer do Reviewer: 1 bug corrigido (link da tarefa buscado fora do escopo do header; o arrasto não fazia nada), 2 riscos menores registrados. Detalhes em backlog/evidencias/issue-142/reviewer.md. make check verde. Pendente: verificação manual do arrasto e aceite visual (#4).

---

author: claude
created: 2026-11-16T14:49:00Z

---

Aceite: aprovada pelo usuário em 16/11/2026 após teste manual do arrasto. Critérios #1 a #3 verificados no teste manual; #4 (capturas desktop e celular) não foi produzido, aceite visual dado pelo usuário.

---
<!-- COMMENTS:END -->
