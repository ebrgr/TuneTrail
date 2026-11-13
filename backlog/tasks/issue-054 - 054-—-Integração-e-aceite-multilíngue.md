---
id: ISSUE-054
title: 054 — Integração e aceite multilíngue
status: Aguardando aceite
assignee: []
created_date: '2026-11-12T23:39:00Z'
labels:
  - i18n
milestone: m-7
dependencies:
  - ISSUE-050
  - ISSUE-051
  - ISSUE-052
  - ISSUE-053
references: []
type: feature
ordinal: 54000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
> Importada de `docs-dev/issues/054-i18n-integracao-aceite.md` — numeração local 054. Original removido em 12/11/2026; consulte o histórico do Git.

**Status:** Implementada — aguardando teste funcional e aceite do usuário · **Depende de:** 050–053 · [Estudo](<../docs/roadmaps/doc-006 - Estudo-—-interface-multilíngue-item-3-do-roadmap.md>)

## Objetivo

Validar a interface multilíngue completa, as traduções e a distribuição antes de apresentar o recurso como concluído.

## Aceite

- [ ] Revisar catálogos, contexto, placeholders, plural, ausência de chaves, fallback, escaping e diagnósticos; automatizar validação em CI.
- [ ] Testar preferências e idiomas simultâneos, `-race`, autenticação, perfis, CSRF, API, fuso, acessibilidade e ausência de JavaScript.
- [ ] Comparar capturas com dados fictícios em desktop/celular, temas claro/escuro, zoom e textos longos; registrar limitações.
- [ ] Validar binário sem CGO, catálogos externos distribuídos, Docker e CI, sem dependência de rede no uso.
- [ ] Documentar como escolher idioma e contribuir com traduções; atualizar changelog e roadmap.

## Resultado

Implementação e verificações registradas em [comparação e validação](../evidencias/i18n/ISSUE_053_054.md). Os itens de aceite permanecem abertos até o teste funcional e a aprovação do usuário.
<!-- SECTION:DESCRIPTION:END -->
