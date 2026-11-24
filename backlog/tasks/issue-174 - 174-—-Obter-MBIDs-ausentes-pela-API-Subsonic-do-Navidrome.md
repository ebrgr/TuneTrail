---
id: ISSUE-174
title: 174 — Obter MBIDs ausentes pela API Subsonic do Navidrome
status: Em andamento
assignee:
  - '@ebrgr'
created_date: '2026-11-20T19:04:00Z'
updated_date: '2026-11-24T16:50:00Z'
labels:
  - ingestao
milestone: m-11
dependencies:
  - ISSUE-134
references:
  - backlog/evidencias/issue-134/PLANO.md
  - backlog/evidencias/issue-174/PLANO.md
  - backlog/evidencias/issue-174/FLUXO.md
  - backlog/evidencias/issue-174/ESTUDO-REUSO-BANCO.md
type: feature
ordinal: 187000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: decisão do usuário em 20/11/2026 sobre o plano da ISSUE-134 (identidade por MBID). Ordem de fontes do MBID de uma escuta: (1) o MBID já salvo no registro ou recebido junto da escuta; (2) quando ausente, consultar a API Subsonic do Navidrome do usuário (getSong/getAlbum/search3; extensões OpenSubsonic quando existirem) e ler os MBIDs de faixa, gravação, álbum e artista que o servidor expõe; (3) quando ausente, a ISSUE-175 consulta o ListenBrainz; (4) se o dado estiver ausente ou o serviço estiver desligado ou indisponível, deixar o MBID em branco, sem pendência nem retry infinito, e resolver a identidade pelos metadados. O MBID obtido de serviço externo é afirmação declarada (não prova): grava-se a fonte e a data, e a promoção a validado segue o contrato da 134. Reaproveitar os parâmetros Subsonic já usados por capas (KOITO_SUBSONIC_PARAMS, internal/images/subsonic.go) e a ISSUE-055/088 (metadados e fonte Subsonic). A consulta ocorre fora do lock de escrita, com timeout, cache e limite de taxa, e nunca bloqueia a admissão. Observação: o formulário de credenciais do Subsonic (user, token, salt) é decisão separada do usuário, ainda pendente.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Com o Subsonic configurado e o MBID ausente, a identidade recebe os MBIDs expostos pelo Navidrome, com fonte e data registradas como declaração
- [ ] #2 Com o serviço desligado, indisponível ou sem o dado, o MBID fica em branco sem pendência, sem retry infinito e sem bloquear a admissão
- [ ] #3 Consulta fora do lock de escrita, com timeout, cache e limite de taxa; credenciais nunca aparecem em log
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: codex
created: 2026-11-24T11:51:00Z

---

Decisão do usuário em 24/11/2026: buscar MBID de gravação e de artista por provedor externo apenas quando o respectivo identificador estiver ausente nos metadados recebidos do Navidrome. Avaliar cada entidade independentemente; preservar o MBID recebido, sem substituí-lo pelo resultado da consulta. A busca de outros metadados segue regras próprias da ISSUE-055.

---

author: ebrgr
created: 2026-11-24T11:55:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-174` na branch `feat/issue-174-obter-mbids-ausentes-pela-api` criada a partir de `main`.

---

author: codex
created: 2026-11-24T11:55:00Z

---

Plano inicial preservado na worktree Orca; implementação pendente.

---

author: codex
created: 2026-11-24T12:41:00Z

---

Implementação: fila independente e limitada para escutas novas, consulta Subsonic com correspondência conservadora e declarações datadas por perfil/instância; revalidação após rede preserva snapshot e identidade. Migração v24, cobertura de migração, escopo, falhas e concorrência, make check e testes de corrida dirigidos passaram; histórico permanece para ISSUE-213 e exportação de declarações de artista requer evolução própria. Pendente parecer do Reviewer e aceite do usuário.

---

author: codex
created: 2026-11-24T13:15:00Z

---

Correções após revisão: consulta de artista considera apenas IDs do próprio crédito; migração v25 desvincula declarações do recibo e preserva escutas e isolamento na exclusão. Arquivo v2 preserva declarações conflitantes e procedência sem promover IDs globais; v1 as omite, e as consultas de enriquecimento da exportação são agrupadas por página. Verificados migração v24→v25, exclusões e privacidade, dois artistas, importação em catálogos distintos, formatos v1/v2, teste de corrida e make check (13 percursos E2E); FLUXO.md descreve regras e justificativas. Pendente nova revisão e aceite.

---

author: codex
created: 2026-11-24T13:22:00Z

---

Revisão de escala: migração v26 projeta créditos com IDs recebidos por recibo e indexa a precedência por perfil, ocorrência e nome. A limpeza visita somente os artistas da ocorrência afetada e usa buscas indexadas do perfil por ocorrência. Testes de migração, precedência, cascatas e planos passaram, incluindo exclusão de 2 mil escutas com 4 mil declarações alheias, make check e testes de corrida dirigidos.

---

author: codex
created: 2026-11-24T14:28:00Z

---

Adaptação do modelo: schema 27 reúne declarações em metadata_values, com uma FK de entidade, perfil, proveniência e escuta validada. A fila durável e a projeção de precedência continuam separadas dos valores. Verificados upgrade v23, migração de declarações, restrições e isolamento, conflitos, remoção, fusões e reversão, arquivo v2, plano de exportação em 300 mil eventos, recuperação após reabrir o banco, teste de corrida e make check com 13 percursos E2E. O estudo registra os tipos futuros e as decisões de revisão de conflito ainda abertas.

---

author: codex
created: 2026-11-24T14:37:00Z

---

Schema v28 exige escuta válida e imutável em metadata_values. A exclusão da escuta remove suas observações, enquanto a remoção do recibo preserva o vínculo existente. Valores antigos sem comprovação ficam em resguardo com motivo e aviso de contagem. Exportação v2 mantém as declarações no evento correspondente, e fusões só transferem valores sustentados pela mesma escuta. Mantida migração aditiva para preservar a compatibilidade dos bancos de trabalho. Passaram os testes de migração, escopo, fusão, exclusão, arquivo e corrida, além de make check com 13 percursos E2E.

---

author: codex
created: 2026-11-24T16:50:00Z

---

Consolidação solicitada das prévias não publicadas: a 024 sobe diretamente do v23 com metadata_values e vínculo obrigatório, projeção de crédito e fila independente. O ensaio em bancos descartáveis confirmou equivalência dos 177 objetos de schema ativos com a cadeia anterior, além de preservação e backup do v23. As prévias 24 a 28 exigem preservação do banco e recuperação em cópia separada. Procedimento documentado e plano da ISSUE-195 atualizado.

---
<!-- COMMENTS:END -->
