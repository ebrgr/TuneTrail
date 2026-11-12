---
id: ISS-051
title: 051 — Internacionalizar interface pública
status: Done
assignee: []
created_date: '2026-11-12T23:39:00Z'
labels:
  - i18n
milestone: m-7
dependencies:
  - ISS-050
references: []
type: feature
ordinal: 51000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
> Importada de `docs-dev/issues/051-i18n-interface-publica-solved.md` — numeração local 051. Original removido em 12/11/2026; consulte o histórico do Git.

**Status:** Fechada e aprovada pelo usuário em 10/11/2026 · **Depende de:** 050 · [Estudo](<../docs/roadmaps/doc-006 - Estudo-—-interface-multilíngue-item-3-do-roadmap.md>)

## Objetivo

Aplicar os idiomas aprovados a perfis, navegação, busca, home, rankings, histórico, artistas, álbuns, faixas e retrospectiva.

## Aceite

- [x] Traduzir textos visíveis, estados vazios, erros, acessibilidade, atributos e mensagens relevantes do JavaScript essencial.
- [x] Preservar nomes de mídia, aliases, URLs, parâmetros, chaves estáveis, API e funcionamento sem JavaScript.
- [x] Conferir fallback, `lang`, usuários e idiomas simultâneos, links e ordenação; testar estados normais e falhos.
- [x] Registrar pendências visuais para comparação final na issue 054.

## Implementação — 10/11/2026

Interface pública, navegação, buscas, rankings, histórico, mídia e retrospectiva passaram a usar os catálogos externos `langs`. As mensagens dos scripts de aparência e exclusão também acompanham o idioma. Nomes de mídia, URLs, filtros e dados de máquina foram preservados. O Advisor aprovou após correção da mensagem de falha de rede.

Datas, nomes de meses, texto relativo, pluralização completa e formatação numérica continuam na [053](<iss-053 - 053-—-Localizar-datas-quantidades-e-unidades.md>); a administração segue na [052](<iss-052 - 052-—-Internacionalizar-administração.md>), e a comparação visual ampla na [054](<iss-054 - 054-—-Integração-e-aceite-multilíngue.md>). [Capturas e evidências](../evidencias/i18n/ISSUE_051.md).

## Encerramento

Issue fechada por solicitação do usuário em 10/11/2026.
<!-- SECTION:DESCRIPTION:END -->
