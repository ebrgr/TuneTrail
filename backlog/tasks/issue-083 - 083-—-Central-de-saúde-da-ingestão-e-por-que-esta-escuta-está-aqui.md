---
id: ISSUE-083
title: 083 — Central de saúde da ingestão e "por que esta escuta está aqui"
status: Done
assignee:
  - '@ebrgr'
created_date: '2026-11-14T00:39:00Z'
updated_date: '2026-11-21T16:11:00Z'
labels:
  - admin
  - ingestao
milestone: m-11
dependencies:
  - ISSUE-082
references:
  - backlog/evidencias/ISSUE-075/reference/admin/ingest-health.html
  - backlog/evidencias/ISSUE-075/reference/admin/listen.html
  - backlog/evidencias/ISSUE-083/PARECER-REVIEWER.md
priority: medium
type: feature
ordinal: 83000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: doc-010 §4.2. `/admin/health` por perfil (último envio, cliente, fila, falhas e motivos, sem tokens) e, na página da escuta, origem, horário recebido, regra aplicada e estado de entrega.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Central de saúde por perfil sem expor segredos
- [x] #2 Testes de isolamento entre perfis
- [x] #3 Explicação por escuta acessível ao dono do perfil e ao root
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-14T03:48:00Z

---

Coordenar com a ISSUE-112 (logs): a central de saúde mostra a contagem de erros e avisos da última hora com link para /admin/logs filtrado.

---

author: claude
created: 2026-11-15T15:18:00Z

---

A página desta tarefa segue o padrão visual comum da remodelação da administração (ISSUE-074) e entra como item da segunda barra (ISSUE-123).

---

author: claude
created: 2026-11-19T16:15:00Z

---

Proposta registrada a pedido do usuário (2026-11-19), a partir do estudo da ISSUE-081 (doc-043): reunir as checagens e revisões da ingestão num lugar próprio, a central de saúde por perfil (/admin/health ou entrada Saúde no menu de gestão do perfil), em vez de espalhá-las pelas telas. Seções sugeridas: (1) Pendências de revisão: fila de duplicatas prováveis e identidades ambíguas geradas pela 081b, com Aceitar como duplicata, Manter como escuta separada e Ver evidência; (2) Últimas decisões da deduplicação: por que esta escuta está aqui, com reason_code, evidência (Δ de tempo, divergência de álbum, ID externo) e rule_version; (3) Fontes e último envio: integração, cliente, horário recebido, estado de entrega e precisão do horário (second ou estimated); (4) Fila e falhas: outbox/relay da 082, tentativas e motivos, sem expor tokens; (5) Verificações automáticas: escutas com horário estimado, lotes rejeitados, relógio adiantado, perfis sem envio há N dias, e os indicadores de qualidade da ISSUE-097. Pontos de atenção: a 081b só deve ativar o modo Revisar (flag por perfil desligada por padrão) quando esta central existir, senão as pendências ficam invisíveis; o acesso passa a ser do dono para o próprio perfil e do root para todos (alinhado à autorização da 160), em vez de só admin; dados sensíveis (tokens, corpo bruto) nunca na tela; a visibilidade da própria central não depende de require_login (é gestão, sempre exige login, ISSUE-161). Sugestão de ajuste de escopo ao replanejar: acrescentar um critério para a fila de revisão e dependência da 081 (081b), e ajustar o AC2 de admin para dono e root.

---

author: ebrgr
created: 2026-11-21T15:32:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-083` na branch `feat/issue-083-central-de-saude-da-ingestao` criada a partir de `main`.

---

author: codex
created: 2026-11-21T15:34:00Z

---

Escopo: painel somente leitura. A fila de revisão manual (081b) e os indicadores da ISSUE-097 ficam pendentes. O checkout não possui painel /admin/logs com métricas por perfil; contagem e link do comentário #1 ficam pendentes. Escutas legadas sem recibo vinculado exibem a origem gravada e indicam os campos indisponíveis.

---

author: codex
created: 2026-11-21T15:43:00Z

---

Verificação: central por perfil e explicação na página de gestão da escuta, com testes de dono, root, visitante e chave de ingestão, incluindo ausência de segredos nas respostas e nos logs do teste. Projeção SQLite conferida nos schemas v11 e v12. make check passou (13/13 jornadas E2E), make ui.reference.check não apontou diferenças e o Graphify foi atualizado. Chrome validou foco, Enter/Espaço e ausência de rolagem horizontal a 320, 768 e 1440 px, nos temas Yuu e Pearl.

---

author: claude
created: 2026-11-21T15:47:00Z

---

Reviewer (claude, 21/11/2026): APROVADO COM RESSALVAS, sem BUG. Verificado: go vet, testes de engine/adminui/adminweb/db/webui e make ui.reference.check verdes; isolamento e ausência de segredos cobertos por TestIngestHealthIsolation. RISCOS: (1) consultas de healthSources/healthDecisions varrem todos os eventos do perfil, sem medição; (2) a Saúde ficou no seletor admin-sections e não na segunda barra citada no comentário #2; (3) AC #2 ainda diz 'admin', mas o acesso é dono e root. Parecer: backlog/evidencias/ISSUE-083/PARECER-REVIEWER.md. Não verificado: tela no navegador.

---

author: claude
created: 2026-11-21T16:02:00Z

---

Ajustes pós-revisão visual (21/11/2026, a pedido do usuário): (1) item Saúde no menu (+), acima de Configuração do servidor para root e junto dos links de conta para donos, com o novo ícone HeartPulse; (2) página de saúde reestilizada como o diálogo de edição de mídia (títulos de seção em caixa alta, linhas preenchidas, botões com borda, hover com transição, layout responsivo verificado em 390 px sem rolagem horizontal); (3) links de conta abaixo dos perfis no diretório espaçados e centralizados. Referência visual ISSUE-075 regenerada. Não verificado: aparência nos demais temas e hover com o mouse (só o estilo no CSS). O ícone HeartPulse foi escrito a partir do Lucide heart-pulse de memória, sem comparar com o pacote.

---

author: claude
created: 2026-11-21T16:08:00Z

---

Aceite: aprovada pelo usuário em 21/11/2026, como central de diagnóstico somente leitura, após ver o preview.

---

author: ebrgr
created: 2026-11-21T16:11:00Z

---

Publicação: versão 0.52 (tag `v0.52`), branch `feat/issue-083-central-de-saude-da-ingestao` integrada ao `main` por fast-forward.

---
<!-- COMMENTS:END -->
