---
id: ISS-035
title: 035 — Refatorar o código de imagens da retrospectiva com imaging
status: Done
assignee: []
created_date: '2026-11-12T23:39:00Z'
labels:
  - imagens
milestone: m-3
dependencies: []
references: []
type: feature
ordinal: 35000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
> Importada de `docs-dev/issues/035-refatoracao-summary-imaging-solved.md` — numeração local 035. Original removido em 12/11/2026; consulte o histórico do Git.

Status: **fechada — resolvida em 06/11/2026**. Issue local, sem publicação.
Depende da conclusão da [034](<iss-034 - 034-—-Distribuir-e-validar-a-aplicação-sem-CGO-libvips.md>).
Fase 2 do [plano](<../docs/estudos/doc-014 - Plano-de-transição-—-imaging-webp-go-pure.md>).

## Trabalho

- Revisar `internal/summary/image.go`, suas funções sem chamadas ativas e
  caminhos relativos; documentar o estado do gerador comentado.
- Substituir a escala manual por imaging com semântica explícita: o código
  antigo força dimensões, enquanto Fit/Fill preservam/cortam proporção.
- Usar composição imaging onde adequada, com testes de coordenadas,
  transparência, pontos negativos, clipping e bounds com origem não zero;
  manter image/draw quando necessário para equivalência e desenho de texto.
- Preservar font/opentype/fixed onde necessários. Imaging não desenha fontes;
  o módulo x/image pode continuar necessário direta e transitivamente.
- Fechar arquivos/font faces e desacoplar assets/fontes do cwd com dependências
  explícitas ou incorporação, respeitando licenças. Reutilizar o componente
  comum para ler imagens, evitando um segundo registro WebP.
- Criar composição fictícia para teste e inspeção visual; não alterar cálculos
  de Summary, retrospectiva HTML, políticas de perfil ou rotas públicas.

## Aceite

Transformações e composição refatoradas com testes/artefatos locais reproduzíveis
e relatório `ISSUE_035.md`, sem necessidade de CGO/libvips. Não ativar GenerateImage
nem criar endpoint/botão de compartilhamento nesta issue. Entregar registro claro
do que permanece para a futura funcionalidade de imagens sociais.

## Entrega

[Relatório, testes e imagens fictícias](../evidencias/transicao-imagens/ISSUE_035.md).
Escala e composição refatoradas, fontes e readers explícitos e decoder/limites
compartilhados; testes Go 1.25, race e corpus passaram. O gerador social continua
desativado e requer especificação própria.
<!-- SECTION:DESCRIPTION:END -->
