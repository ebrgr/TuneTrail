# 052 — Internacionalizar administração

**Status:** Implementada — aguardando aceite · **Depende de:** 050 · [Estudo](../ROADMAP_03_UI_MULTILINGUE.md)

## Objetivo

Traduzir o fluxo administrativo Go com os mesmos catálogos e escolha de idioma da UI pública.

## Aceite

- [x] Cobrir login, perfis, importação/exportação, chaves, relay, configuração, edição, mesclagem e remoção, com erros, avisos e confirmações.
- [x] Cobrir acessibilidade e mensagens dos scripts essenciais; idioma igual em login, páginas e respostas de erro.
- [x] Preservar segurança, CSRF, no-store, permissões, formatos de máquina e valores de usuário; testar sucesso, falha, sem JavaScript e concorrência.
- [x] Não traduzir chaves de ambiente nem dados de catálogo; registrar pendências visuais na issue 054.

## Resultado

Administração, mensagens e rótulos acessíveis usam os catálogos externos em `langs`. O idioma segue a preferência do navegador e a escolha nas configurações. [Validação e comparações](../comparisons/i18n/ISSUE_052.md). Datas e plurais permanecem na issue 053; a revisão visual ampla permanece na 054.
