---
id: ISS-049
title: 049 — Associação residual de artista em álbum após edição de faixa
status: Aberta
assignee: []
created_date: '2026-11-12T23:39:00Z'
labels:
  - catalogo
  - banco
milestone: m-6
dependencies: []
references: []
priority: high
type: bug
ordinal: 49000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
> Importada de `docs-dev/issues/049-associacao-residual-artista-album.md` — numeração local 049. Original removido em 12/11/2026; consulte o histórico do Git.

**Status:** Aberta · **Identificada:** 09/11/2026

## Problema reproduzido

A página do artista Engenheiros do Hawaii (ID 1) lista o álbum **1**, dos
Beatles (ID 23), apesar da remoção da associação do artista na faixa **Love Me
Do** (ID 89). A faixa já não aparece entre as faixas do artista; o álbum ainda
aparece na seção de álbuns associados. Verificado pelas páginas públicas, sem
alterar o banco do usuário.

Ao associar um artista à faixa, `UpdateTrack` também cria uma relação
`artist_releases` com o álbum. Ao remover o artista da faixa, a relação da faixa
é excluída, mas a do álbum é preservada. As consultas de álbuns por artista usam
essa segunda relação. Alias de faixa e relação entre artistas são dados distintos.

## Risco e direção

As relações existentes de álbum não registram procedência: uma associação
secundária pode ser crédito independente e legítimo, ou ter sido criada por
uma faixa. Não é seguro apagar automaticamente todas as relações sem faixa
correspondente; álbuns podem ter créditos próprios e catálogos incompletos.
Uma trigger também apaga o álbum se a última associação for removida.

Estudar separação entre créditos explícitos e relações derivadas das faixas,
inclusive o caso em que ambos existem simultaneamente. Preservar créditos
legados cuja origem não pode ser determinada; propor revisão ou remoção manual
específica e transparente. Qualquer mudança de esquema deve ser planejada e
validada com migração, backup e compatibilidade do SQLite v3, sem alterar
migrations consolidadas retroativamente.

## Critérios de aceite

- [ ] Adicionar e remover artista de faixa atualiza relações derivadas, sem
  deixar um álbum indevido na página do artista.
- [ ] Preservar créditos explícitos, primários, legados ambíguos e relações
  ainda justificadas por outra faixa; contemplar origem dupla e álbuns parciais.
- [ ] Tratar dados antigos com revisão explícita, sem limpeza global automática
  e sem exclusão acidental de álbum, faixas, escutas ou dados de outros perfis.
- [ ] Cobrir persistência, transação, rollback, páginas e rankings por artista,
  testes de regressão e integridade SQLite; registrar procedimento de recuperação.
- [ ] Atualizar changelog e documentação após a correção.

Advisor/Sol apontou que não se pode inferir a procedência com os campos atuais.
<!-- SECTION:DESCRIPTION:END -->
