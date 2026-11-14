---
id: ISSUE-050
title: 050 — Fundação e escolha de idioma
status: Done
assignee: []
created_date: '2026-11-12T23:39:00Z'
labels:
  - i18n
milestone: m-7
dependencies: []
type: feature
ordinal: 115000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
> Importada de `docs-dev/issues/050-i18n-fundacao-idioma-solved.md` — numeração local 050. Original removido em 12/11/2026; consulte o histórico do Git.

**Status:** Fechada e aprovada pelo usuário em 10/11/2026 · [Estudo](<../docs/roadmaps/doc-006 - Estudo-—-interface-multilíngue-item-3-do-roadmap.md>)

## Objetivo

Provar a escolha da biblioteca e preparar idioma por navegador, compartilhado por UI pública e administração.

## Aceite

- [x] Comparar `go-i18n/v2` com `x/text/message` em prova de conceito; fixar versão compatível com Go/Docker/CI e documentar dependências, extração, licença e custos.
- [x] Separar identificadores de cards e períodos da apresentação antes de traduzir.
- [x] Carregar catálogos externos no diretório `langs`, validar chaves/placeholders e resolver idioma por requisição com inglês de fallback.
- [x] Seletor acessível sem JavaScript, cookie por navegador, `Accept-Language`, retorno local e proteções de origem aplicáveis.
- [x] Não compartilhar estado mutável do idioma nem interpolar HTML confiável; testar simultaneidade, escaping, tags inválidas e idiomas não suportados.
- [x] Preservar SQLite v3, perfil, API, datas e fuso; documentar decisões de `pt` versus `pt-BR` e cobertura da primeira fatia.

## Implementação — 10/11/2026

Catálogos JSON externos em `langs`, carregados no início com `go-i18n/v2` 2.5.1 e `x/text/language`; não são embutidos no binário. A seleção vale por navegador, com fallback inglês, e funciona sem JavaScript nas páginas públicas e administrativas. A primeira fatia traduz a seleção de perfis e o seletor; as demais telas e plurais de catálogo permanecem nas issues 051–053. A preferência não altera SQLite, perfil, fuso ou API. [Capturas e validações](../evidencias/i18n/ISSUE_050.md).

## Ajuste de localização — 10/11/2026

O seletor foi movido para os diálogos de configurações da interface pública e da administração. Sem JavaScript, uma seção recolhida de configurações oferece o mesmo formulário. O idioma continua automático pelo navegador, com escolha manual armazenada em cookie; não é salvo no TOML. [Capturas atualizadas](../evidencias/i18n/ISSUE_050.md).

## Encerramento

Issue fechada por solicitação do usuário em 10/11/2026.
<!-- SECTION:DESCRIPTION:END -->
