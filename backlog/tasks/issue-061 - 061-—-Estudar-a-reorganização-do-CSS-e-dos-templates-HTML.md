---
id: ISSUE-061
title: 061 — Estudar a reorganização do CSS e dos templates HTML
status: Done
assignee: []
created_date: '2026-11-13T23:47:00Z'
updated_date: '2026-11-14T15:02:00Z'
labels:
  - ui
milestone: m-10
dependencies: []
references:
  - backlog/evidencias/issue-061/fontes-retrospectiva.md
  - backlog/evidencias/issue-061/parecer-claude-opus.md
documentation:
  - doc-031
type: spike
ordinal: 1000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Estudo para deixar CSS e templates HTML legíveis e fáceis de manter, sem Tailwind (ISSUE-058, decision-006) e sem build de front-end. Decisão do usuário em 14/11/2026: reunir todos os arquivos web em assets/ na raiz (HTML em assets/templates/), ler do disco em produção e desenvolvimento, usar fontes locais e planejar embed no binário somente como opção futura. Inclui investigação das diferenças tipográficas entre as retrospectivas de referência e TuneTrail. Estudo vigente: doc-031; implementação distribuída em ISSUE-065 a ISSUE-070.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Responder se CSS e templates são compilados no binário ou lidos do disco
- [x] #2 Medir a situação atual (tamanhos, linhas, pontos de quebra, dependências de caminho)
- [x] #3 Comparar localizações (internal, raiz embutido, raiz externo, híbrido) e recomendar
- [x] #4 Propor organização do CSS e dos templates com método de verificação por fase
- [x] #5 Dividir a implementação em fases que viram tarefas separadas
- [x] #6 Registrar decisão de diretório único assets/ e leitura do disco, fontes locais e opção embed futura.
- [x] #7 Investigar fontes existentes e diferenças tipográficas entre as duas retrospectivas indicadas, com evidência e encaminhamento.
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Decisão explícita de 14/11/2026: tudo em assets/ (inclusive HTML em assets/templates/), leitura do disco e fontes locais; embed será opção futura. Estudo doc-031 reescrito e 065/066/070 alinhadas. Inspeção de ambas as retrospectivas confirmou diferenças de peso, caixa e tamanho; dois TTF estáticos existentes não cobrem toda a tipografia. Não houve implementação de runtime.
<!-- SECTION:NOTES:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-13T23:47:00Z

---

Verificação: estudo doc-031 escrito a partir de leitura de internal/webui/preview.go:21, internal/adminui/admin.go:35, internal/webui/assets.go, internal/i18n/i18n.go (precedente langs/), Dockerfile e medição de todos os templates e CSS (tamanho, linhas, maior linha, media queries). Sem mudança de código. Aguarda decisões do usuário: embutido+dev (recomendado) ou externo; nomes templates/ e assets/; fonte local na mesma leva ou separada.

---

author: claude
created: 2026-11-14T00:39:00Z

---

Plano TuneTrail 1.0 (doc-032): as fases do doc-031 viraram ISSUE-065 a ISSUE-070. Pontos de quebra alinhados ao doc-010: 40, 64 e 90rem.

---

author: Codex
created: 2026-11-14T14:22:00Z

---

Verificação: estudo atualizado após decisões do usuário (tudo em assets/, disco, fontes locais, embed futuro). CSS/estilos computados das duas URLs comparados e inventário SFNT dos TTF realizado; evidência anexada. 065/066/070 refinadas; 066 será mudança funcional de distribuição com versão. CLI Claude tentado, mas sem login; Advisor @Sol aprovou contrato de snapshot/root/deploy/rotas restritas. Sem alteração de runtime ou nova suíte Go, pois a entrega é estudo/documentação. Aguarda aceite do estudo revisado.

---

author: Codex
created: 2026-11-14T14:33:00Z

---

Advisor solicitado em 14/11/2026: invocação Claude Code CLI funciona fora da sandbox. Dentro da sandbox auth status indicava false; fora reconheceu sessão existente. Smoke real Opus respondeu ADVISOR_OK; revisão real Opus effort medium, Read/Grep/Glob, concluiu com código 0 e parecer REPROVADO COM RESSALVAS. Parecer completo anexado. Pontos: alvo tipográfico precisa confirmação após comparação controlada; ACs da 065 devem incluir tipografia; detalhar hashes de fontes, manifesto/rotas, permissões, confinamento de symlinks e smoke de renderização. Demais observações e limites constam na evidência; sugestões não aplicadas ao plano nesta etapa. Procedimento local .agents/advisor.md corrigido com separador -- e diagnóstico da sandbox, mantendo arquivo fora do Git. Status preservado.

---

author: claude
created: 2026-11-14T15:00:00Z

---

Resolução do parecer do Advisor (Claude, 14/11/2026, com decisões do usuário): (1) dados da instalação na evidência — dispensado pelo usuário (endereço de rede local genérico); (2) alvo tipográfico — decisão do usuário: igualar o Koito nos pesos e estilos das fontes (família, peso, caixa, itálico); tamanhos só após a medição da 065 em três larguras e confirmação; registrado na ISSUE-070 (AC #4); (3) referência tipográfica e seção de origem na 065 — já resolvidos pelo Codex (AC #4 da 065, aceita). Requisitos 4–10 registrados: hash nos url() do CSS (068, 070, 106), manifesto por extensão com licenças fora das rotas, root só leitura (066, 102), confinamento de symlinks, gate de renderização no CI e templates público/admin separados, contrato de langs/ (066), Reserved Font Name da OFL (070). Item 11: a comparação de localizações (AC #3) que saiu do doc-031 reescrito permanece no histórico do Git, commit 8e772f3. Nenhuma ressalva bloqueia o aceite desta tarefa.

---

author: claude
created: 2026-11-14T15:02:00Z

---

Aceite: aprovada pelo usuário em 14/11/2026, com as ressalvas do Advisor resolvidas conforme o comentário anterior.

---
<!-- COMMENTS:END -->
