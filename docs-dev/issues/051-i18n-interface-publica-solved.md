# 051 — Internacionalizar interface pública

**Status:** Fechada e aprovada pelo usuário em 10/11/2026 · **Depende de:** 050 · [Estudo](../ROADMAP_03_UI_MULTILINGUE.md)

## Objetivo

Aplicar os idiomas aprovados a perfis, navegação, busca, home, rankings, histórico, artistas, álbuns, faixas e retrospectiva.

## Aceite

- [x] Traduzir textos visíveis, estados vazios, erros, acessibilidade, atributos e mensagens relevantes do JavaScript essencial.
- [x] Preservar nomes de mídia, aliases, URLs, parâmetros, chaves estáveis, API e funcionamento sem JavaScript.
- [x] Conferir fallback, `lang`, usuários e idiomas simultâneos, links e ordenação; testar estados normais e falhos.
- [x] Registrar pendências visuais para comparação final na issue 054.

## Implementação — 10/11/2026

Interface pública, navegação, buscas, rankings, histórico, mídia e retrospectiva passaram a usar os catálogos externos `langs`. As mensagens dos scripts de aparência e exclusão também acompanham o idioma. Nomes de mídia, URLs, filtros e dados de máquina foram preservados. O Advisor aprovou após correção da mensagem de falha de rede.

Datas, nomes de meses, texto relativo, pluralização completa e formatação numérica continuam na [053](053-i18n-datas-numeros.md); a administração segue na [052](052-i18n-administracao.md), e a comparação visual ampla na [054](054-i18n-integracao-aceite.md). [Capturas e evidências](../comparisons/i18n/ISSUE_051.md).

## Encerramento

Issue fechada por solicitação do usuário em 10/11/2026.
