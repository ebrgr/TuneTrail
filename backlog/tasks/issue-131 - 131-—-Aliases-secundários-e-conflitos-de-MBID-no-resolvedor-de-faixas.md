---
id: ISSUE-131
title: 131 — Aliases secundários e conflitos de MBID no resolvedor de faixas
status: Done
assignee:
  - '@ebrgr'
created_date: '2026-11-15T16:43:00Z'
updated_date: '2026-11-16T16:26:00Z'
labels:
  - catalogo
  - ingestao
milestone: m-11
dependencies: []
references:
  - backlog/evidencias/issue-131/verificacao.md
  - backlog/evidencias/issue-131/parecer-reviewer.md
documentation:
  - doc-036
priority: medium
type: bug
ordinal: 144000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Recorte 1 do estudo da ISSUE-116 (doc-036). No resolvedor atual, um alias secundário cria faixa nova em vez de achar a existente (Koito #369), e títulos iguais com MusicBrainz IDs diferentes se confundem (PR #315, #214). Pequeno e sem migração.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Alias secundário no mesmo contexto reutiliza o ID existente
- [x] #2 Alias ambíguo produz decisão explícita, sem escolha silenciosa
- [x] #3 MBIDs diferentes com títulos iguais nunca retornam o primeiro com o MBID do segundo; retorno e banco concordam
- [x] #4 Testes cobrem artista adicional e ordem dos artistas
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-15T16:59:00Z

---

Reclassificada a pedido do usuário (2026-11-15): correção adiada, prioridade alta → média. Continua obrigatória antes da ISSUE-132, que já depende desta tarefa.

---

author: advisor
created: 2026-11-16T16:02:00Z

---

Estudo Advisor sobre 63aac9c: alias secundário ignorado em internal/db/sqlite/track.go:118; QueryRow em :132 escolhe silenciosamente entre candidatas; internal/catalog/associate_track.go:76 pode devolver MBID diferente do persistido. Correção proposta sem migração: consultar todos os aliases exatos no mesmo lançamento, deduplicar artistas preservando inclusão e independência da ordem, detectar ambiguidade e conflito de identidade, propagar erros em ambos os caminhos de título e preencher MBID ausente de forma transacional. Atualizar DoNotOverwriteMbzIDs e cobrir aliases ambíguos, MBIDs conflitantes, artistas adicionais/reordenados/repetidos, falhas de banco e concorrência. Arquivos-alvo: internal/db/opts.go, internal/db/errors.go, internal/db/sqlite/track.go, internal/catalog/associate_track.go e testes correspondentes. Restrição: preservar título principal e modelo atual, sem migração ou mudanças de caixa/remaster. Ponto de decisão antes da implementação: recusar conflito impede salvar a escuta, interrompe importação Last.fm e atualmente resulta em HTTP 500 no ListenBrainz; preservação durável de pendências pertence à ISSUE-132+. Não foram executados testes nem alterado código. ai-memory indisponível nesta sessão.

---

author: ebrgr
created: 2026-11-16T16:19:00Z

---

Início: branch `fix/issue-131-aliases-secundarios-e-conflitos-de` criada a partir de `main`.

---

author: codex
created: 2026-11-16T16:23:00Z

---

Implementação: aliases exatos no mesmo lançamento, artistas sem dependência de ordem/repetição, erros explícitos para ambiguidade e MBIDs conflitantes, preenchimento de identidade transacional sem sobrescrita. Verificação: quatro regressões falham no código anterior; make check e race nos pacotes catalog/sqlite verdes. Recorte mantém rejeição de escutas conflitantes sem fila de pendências (ISSUE-132+).

---

author: codex
created: 2026-11-16T16:23:00Z

---

Reviewer: APROVADO. Diff da implementação 7ee2d64 revisado diretamente, incluindo erros nos dois caminhos de título, deduplicação de artistas, fechamento de rows antes de outras consultas e preenchimento transacional do MBID. Impacto consultado no graphify e revisão deslop realizada. Sem itens obrigatórios.

---

author: ebrgr
created: 2026-11-16T16:24:00Z

---

Verificação: make check verde, quatro testes de regressão falham no código anterior, testes race de catalog/sqlite verdes, rollback e concorrência cobertos; Reviewer APROVADO. Sem migração. Conflitos recusam a escuta; pendências duráveis ficam na ISSUE-132+. Evidências em backlog/evidencias/issue-131/.

---

author: codex
created: 2026-11-16T16:26:00Z

---

Aceite: aprovada pelo usuário em 16/11/2026. Pendências sobre preservação de escutas conflitantes e comportamento das fontes encaminhadas para definição de contrato na ISSUE-132, conforme solicitado.

---
<!-- COMMENTS:END -->
