---
id: ISSUE-144
title: 144 — Observações e pendências das ISSUE-139 a 143
status: Done
assignee:
  - '@ebrgr'
created_date: '2026-11-16T00:54:00Z'
updated_date: '2026-11-16T14:59:00Z'
labels:
  - ui
  - admin
  - processo
milestone: m-10
dependencies: []
references:
  - backlog/evidencias/issue-141/parecer-coordenador.md
  - backlog/evidencias/issue-143/parecer-coordenador.md
priority: low
type: chore
ordinal: 157000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Lista de consulta posterior, reunida em 16/11/2026 a partir dos pareceres do coordenador das ISSUE-141 e 143 e do uso do fluxo de publicação. Nenhum item bloqueou o aceite; cada um pode virar tarefa própria quando for priorizado. ISSUE-143: (1) com contagem 0 a janela de exclusão por perfil ainda mostra o botão de excluir; desabilitar o botão e avisar que não há escutas a remover. (2) O código da exclusão global do catálogo (revisão e confirmação de delete) continua no servidor sem alcance pela interface; remover numa tarefa de limpeza, com testes. ISSUE-141: (3) depois de apagar a última escuta de uma faixa, álbum ou artista a janela mostra o estado vazio, mas recarregar a página dá 404; decidir se administrador deve ver a página vazia também em GET normal. (4) Vocabulário em português: a janela diz reprodução e outros pontos dizem escuta; padronizar. ISSUE-140: (5) a página de imagem sem JavaScript (admin.css) não usa o layout em coluna do formulário de troca de imagem do diálogo; aplicar o mesmo layout. Processo: (6) make issue.publish lê o status da tarefa no checkout principal, mas com worktrees o aceite fica na branch da worktree; o publish falha ('está Aberta') ou quebra no commit do aceite. Ler a tarefa na branch ou na worktree, e tratar o aceite já commitado. (7) O script externo analisar-tunetrail.sh (fora do repositório, em ~/.local/share/tunetrail-graphify) só aceita .git como diretório e rejeita worktrees, onde .git é arquivo; trocar a validação de -d para -e (decisão do usuário, ferramenta externa).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Cada item acima foi convertido em tarefa própria, descartado com motivo ou implementado
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: ebrgr
created: 2026-11-16T14:55:00Z

---

Início: branch `chore/issue-144-observacoes-e-pendencias-das-issue` criada a partir de `main`.

---

author: ebrgr
created: 2026-11-16T14:58:00Z

---

Verificação: item 6 implementado em scripts/issue.py (publish lê o registro na worktree ou na branch e commita aceite pendente da worktree), com 3 testes novos que falham no código antigo e passam no novo (35/35 em scripts/test_issue.py); make check verde. Item 7: analisar-tunetrail.sh passou a validar .git com -e (backup .bak-issue144 ao lado do script). Itens 1 a 5 convertidos nas ISSUE-148 (contagem 0), 149 (exclusão global morta), 150 (página vazia para o admin, decisão do usuário), 151 (vocabulário escuta) e 152 (layout da página de imagem sem JS).

---

author: claude
created: 2026-11-16T14:59:00Z

---

Aceite: aprovada pelo usuário em 16/11/2026.

---
<!-- COMMENTS:END -->
