# 054 — Integração e aceite multilíngue

**Status:** Implementada — aguardando teste funcional e aceite do usuário · **Depende de:** 050–053 · [Estudo](../ROADMAP_03_UI_MULTILINGUE.md)

## Objetivo

Validar a interface multilíngue completa, as traduções e a distribuição antes de apresentar o recurso como concluído.

## Aceite

- [ ] Revisar catálogos, contexto, placeholders, plural, ausência de chaves, fallback, escaping e diagnósticos; automatizar validação em CI.
- [ ] Testar preferências e idiomas simultâneos, `-race`, autenticação, perfis, CSRF, API, fuso, acessibilidade e ausência de JavaScript.
- [ ] Comparar capturas com dados fictícios em desktop/celular, temas claro/escuro, zoom e textos longos; registrar limitações.
- [ ] Validar binário sem CGO, catálogos externos distribuídos, Docker e CI, sem dependência de rede no uso.
- [ ] Documentar como escolher idioma e contribuir com traduções; atualizar changelog e roadmap.

## Resultado

Implementação e verificações registradas em [comparação e validação](../comparisons/i18n/ISSUE_053_054.md). Os itens de aceite permanecem abertos até o teste funcional e a aprovação do usuário.
