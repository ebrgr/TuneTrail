# ISSUE-061 — Fontes e retrospectivas

Inspeção de 14/11/2026. Escopo somente leitura, sem alterar temas, páginas, arquivos de runtime ou dados. Chrome, viewport nativo de aproximadamente 1487 × 841 px. A referência está em tema claro e a instalação em tema escuro; esta sessão compara estilos computados, não equivalência visual pixel a pixel.

## Páginas verificadas

- Referência Koito 0.3.2: `http://127.0.0.1:4111/rewind`.
- TuneTrail 0.10: `http://192.168.88.33:4110/evertonberger/rewind`.

As duas páginas foram abertas e verificadas pelo navegador. Foram lidos os estilos computados dos elementos, links de CSS e os CSS locais por HTTP. Os nomes das famílias em `getComputedStyle` representam a declaração CSS; não houve medição da fonte efetivamente usada para cada glifo. Não há evidência nesta sessão para atribuir a diferença a falha de download ou a uma fonte de sistema específica.

## Comparação dos estilos computados

| Elemento | Referência | TuneTrail |
| --- | --- | --- |
| Corpo | Jost, Inter, fontes de sistema; 16 px / 400 | Jost, Inter, fontes de sistema; 16 px / 400 |
| Título da retrospectiva | League Spartan; 36 px / 600; altura 40 px | League Spartan; 42 px / 600; altura 54,6 px |
| Nome do artista vencedor | League Spartan; 36 px / 600; sem transformação de caixa; altura 40 px | League Spartan; 36 px / 400; `uppercase`; altura 39,6 px |
| Rótulo de seção | League Spartan; 16 px / 400; `uppercase`; altura 24 px | League Spartan; 18 px / 600; `uppercase`; altura 27 px |
| Nome do álbum vencedor | League Spartan; 24 px / 600 no exemplo com título longo | League Spartan; 36 px / 400; `uppercase` no exemplo atual |
| Ano | Jost/Inter/sistema; 20 px / 500 | League Spartan; 20 px / 600 |

O tamanho do álbum na referência pode depender do comprimento do título. A implementação deve conferir o componente original antes de fixar um tamanho uniforme. Idiomas, comprimento dos títulos e temas também diferem nos exemplos observados.

## Causa confirmada no CSS

`internal/webui/static/ui.css:61–77` define League Spartan nos headings, peso 600 em h1/h2 e **peso 400 + uppercase em todos os h3**. Em `:1163`, `.rewind-top h3` troca tamanho e altura, mas herda peso e uppercase. Na referência, os vencedores são h5 com peso 600 e sem uppercase. Essa diferença explica a aparência mais fina e em caixa alta no TuneTrail mesmo com a mesma família declarada.

O título usa `.rewind-content h1` com máximo 42 px (`:1153`), enquanto a referência mostra 36 px. `.rewind-group h2` usa 18 px (`:1158`), herdando peso 600, contra 16 px / 400 na referência. Portanto, somente trocar Google Fonts por arquivos locais não igualará as retrospectivas. A correção deve ser específica das classes da retrospectiva, preservando a semântica HTML e sem alterar globalmente h3 de outras páginas.

## Origem das fontes

- Referência: `root-BAGEcYhn.css` importa Jost e League Spartan pelo Google Fonts com pesos variáveis 100–900 e Jost normal/itálico. O HTML também liga o CSS de Inter do Google Fonts.
- TuneTrail: `live.html` e `page.html` ligam Jost, League Spartan e Inter pelo Google Fonts. `internal/adminui/templates/shell.html` liga Jost e League Spartan com pesos 400/500/600/700.
- Nenhum dos CSS locais consultados define `@font-face` local. Copiar os TTF para a imagem não os torna fontes web sem rota e `@font-face` correspondentes.

Hashes SHA-256 dos CSS lidos por HTTP:

| CSS | SHA-256 |
| --- | --- |
| Referência `root-BAGEcYhn.css` | `de9ec69f507b89b54014a7b85ae8e4f21b689851637ea193387bacf69a9dfe4f` |
| Referência `useTheme-CjQzxIoJ.css` | `352072ca711a8a9154465453f76af4435a3feb2e24340ebac42ee81d657ecb80` |
| TuneTrail `ui.css` | `65e367f987cbf45b9398eb5d9f2db76ba5b24406a1eef0d8175e8283ec5ff83e` |

## Inventário local

Busca em arquivos do projeto, sem diretórios privados ou dependências externas. Metadados lidos diretamente das tabelas SFNT `name`, `OS/2` e `fvar` dos TTF.

| Arquivo | Família / estilo | Peso OS/2 | Variável | Tamanho |
| --- | --- | ---: | --- | ---: |
| `assets/Jost-Regular.ttf` | Jost Regular | 400 | Não | 61.524 bytes |
| `assets/LeagueSpartan-Medium.ttf` | League Spartan Medium | 500 | Não | 57.540 bytes |

Total: **119.064 bytes**. Os dois arquivos declaram SIL Open Font License 1.1 nos metadados 13/14. Não foi encontrado texto OFL separado para essas fontes, Inter, WOFF ou WOFF2 no inventário do projeto.

Os arquivos atuais não cobrem, sozinhos, os pesos/itálicos da interface nem os pesos 400/600 de League Spartan medidos nas retrospectivas. Não declarar um TTF estático como se fosse variável 100–900. O plano da ISSUE-070 deve obter os arquivos necessários e suas licenças de distribuição, registrar origem/hash/peso/estilo e verificar acentos portugueses e fonte realmente renderizada. A existência de TTFs antigos não justifica removê-los sem conferir consumidores; o protótipo de imagem em `internal/summary/image.go` está desativado e recebe bytes de fonte do chamador.

## Encaminhamento

A ISSUE-065 deve guardar a referência controlada. A ISSUE-070 fará fontes locais e ajuste tipográfico da retrospectiva como mudança visual explícita, com capturas antes/depois no mesmo tema, viewport e conjunto de dados fictícios. Testar ausência de Google Fonts, fontes servidas pelo próprio servidor e funcionamento com rede externa bloqueada.
