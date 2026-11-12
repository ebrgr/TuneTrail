# 053 — Localizar datas, quantidades e unidades

**Status:** Implementada — aguardando teste funcional e aceite do usuário · **Depende de:** 050 · [Estudo](../ROADMAP_03_UI_MULTILINGUE.md)

## Objetivo

Formatar quantidades, datas, meses, tempo relativo e descrições de gráficos de acordo com idioma e locale aprovados.

## Aceite

- [ ] Tratar plural e parâmetros em mensagens completas; verificar singular, plural, zero e traduções em pt-BR/en.
- [ ] Localizar datas, horários, intervalos, números e rótulos acessíveis sem alterar timestamps, UTC, fuso, DST ou primeiro dia da semana existentes.
- [ ] Preservar ISO, dados `data-*`, SVG, URLs e formatos de entrada; garantir compatibilidade sem JavaScript.
- [ ] Testar limites de calendário, tempo relativo e formatos de máquina, inclusive gráficos e retrospectiva.

## Resultado

Implementação e verificações registradas em [comparação e validação](../comparisons/i18n/ISSUE_053_054.md). Os itens de aceite permanecem abertos até o teste funcional e a aprovação do usuário.
