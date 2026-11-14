---
id: doc-035
title: Estudo da ISSUE-067 — head único, nomes de blocos e linhas curtas
type: other
created_date: '2026-11-15T00:30:00Z'
updated_date: '2026-11-15T00:30:00Z'
tags:
  - estudo
---
**Tarefa:** ISSUE-067 · **Base:** `main` em `8700fbb`, TuneTrail 0.14 · **Data:** 14/11/2026.

A 066 já tinha reformatado a maior parte dos templates (uma tag estrutural por linha). Esta etapa resolve o que faltava: três `<head>` copiados, nomes de blocos sem padrão e linhas longas, sem mudar o que o navegador mostra.

## Situação encontrada

| Ponto | Antes |
|---|---|
| `<head>` | Três cópias: `live.html` (páginas públicas), `page.html` (prévia) e `admin-head` (administração, em outro conjunto de templates) |
| Barra lateral | Duas: `sidebar` (público e prévia) e `admin-nav` (administração) |
| Nomes | 41 blocos misturando `live-*`, nomes de arquivo (`live.html`, `admin.html`) e nomes soltos (`home`, `media`, `profiles`) |
| Linhas acima de 160 caracteres | 48 nos templates e 11 no CSS (até 3.195 caracteres no `admin.css`) |

## Decisões

1. **Um head para os dois conjuntos.** `assets/templates/shared/head.html` define `partial/head`. O manifesto monta o mesmo arquivo como `public/templates/head.html` e `admin/templates/head.html`; `validEntry` passou a aceitar `templates/shared/` só para entradas de template. A página chama `{{template "partial/head" (head "live"|"preview"|"admin" título)}}`. A função `head` (`internal/webassets/head.go`) recusa tipos desconhecidos. As chamadas `asset` continuam com nomes literais, como exige o validador da 106.
2. **Prefixos de bloco:** `page/...` (documentos e seções), `partial/...` (componentes) e `admin/...` (administração). Os blocos só da prévia ganharam `preview-` no nome (`page/preview-home`, `partial/preview-media`) para não colidir com os da página real.
3. **Linhas curtas sem mudar a saída.** CSS quebrado depois de `{`, `;` e `}` e, numa declaração longa, nas vírgulas do primeiro nível da função. Templates quebrados só onde a quebra não gera texto: entre duas ações coladas ou entre `>` e uma ação, com `{{-`; dentro de uma ação; entre atributos de uma tag. `scripts/check_web_style.py` entra no `make ui.check`. Exceções: linha com um único atributo sem espaços no valor (a URL do Google Fonts, que sai na 070) e linha com uma única string de tradução.
4. **Referência nova** em `backlog/evidencias/issue-067/reference/`; as da 065, 066 e 106 ficam como histórico.

## Diferenças intencionais no HTML

| Onde | Mudança | Efeito |
|---|---|---|
| 9 páginas da administração | Fontes carregadas como no público: `preconnect` e a família variável completa, no lugar dos pesos 400–700 | Mesmas famílias e pesos usados; uma conexão antecipada a mais. Some na 070 (fontes locais) |
| 2 páginas da prévia | Atributos do favicon na mesma ordem das outras páginas | Nenhum |
| Todas | Espaços entre tags e dentro de tags | Nenhum |

## Verificação

- Comparação normalizada (`backlog/evidencias/issue-067/comparar_normalizado.py`): das 113 saídas, todas as páginas públicas são equivalentes à referência da 106; só as 11 páginas acima diferem, pelos motivos da tabela.
- Comparação estrita da etapa de linhas curtas (espaços preservados, só colapsados): 0 diferenças em 113 arquivos. Nenhum espaço visível apareceu ou sumiu.
- CSS: `ui.css` e `admin.css` equivalentes ao `main` depois de remover espaços e comentários.
- Capturas: 138 de 138 idênticas byte a byte (23 páginas × 6 larguras, fontes reais); ver `backlog/evidencias/issue-067/capturas.md`.
- `make check`: 12/12 jornadas, referência nova sem diferenças, verificador de estilo verde.

## Barra lateral: pendente de decisão

As duas barras não são equivalentes: na administração a busca é um link (no público, um diálogo), os ícones são SVG escritos no template (no público, a função `icon`), o avatar usa campos de outro modelo e o botão de configurações abre outro diálogo. Unificá-las sem mudar o HTML só esconderia as duas dentro de um `if`; unificá-las de verdade muda o comportamento da administração, o que contraria o critério 1 desta tarefa. As ISSUE-109 (selo de sessão de administrador) e ISSUE-110 (ícone de tema e menu de administração) redesenham essa barra.

**Proposta:** manter as duas nesta tarefa, já com os nomes `partial/sidebar` e `admin/sidebar`, e mover o critério "barra lateral definida uma única vez" para a ISSUE-110. Depende da decisão do usuário.
