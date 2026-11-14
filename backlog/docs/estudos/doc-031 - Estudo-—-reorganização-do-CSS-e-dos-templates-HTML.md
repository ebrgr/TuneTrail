---
id: doc-031
title: Estudo — reorganização do CSS e dos templates HTML
type: other
created_date: '2026-11-13T23:46:00Z'
tags:
  - estudo
---

**Tarefa:** ISSUE-061 · **Substitui:** ISSUE-058 (Tailwind não será adotado, decision-006) · **Data:** 13/11/2026

## Pergunta

Como deixar o CSS e os templates HTML legíveis e fáceis de manter, sem Tailwind e sem etapa de build, e se eles devem ficar em `/templates` e `/assets` na raiz do repositório.

## Resposta curta

- **Hoje os arquivos são compilados no binário.** `internal/webui/preview.go:21` embute `templates/*.html static palettes.json` e `internal/adminui/admin.go:35` embute `templates/*.html` com `go:embed`. Nada de CSS ou template é lido do disco em produção. Só os catálogos de tradução (`langs/`) são externos.
- **O problema de legibilidade não é a localização, é a escrita.** Os templates estão em linhas únicas de até 2.261 caracteres; `admin.css` é uma linha só de 3.195; `ui.css` tem 1.199 linhas organizadas pela história dos roadmaps, não por função.
- **Recomendação:** mover para `templates/` e `assets/` na raiz, **continuar embutindo no binário** (um pacote Go na raiz faz o `go:embed`), com uma variável de desenvolvimento para ler do disco sem recompilar. Depois, reformatar os templates e dividir o CSS em camadas, em fases verificáveis.

## Situação atual (medida em 13/11/2026)

### Templates

| Arquivo | Bytes | Linhas | Maior linha |
| --- | ---: | ---: | ---: |
| `internal/webui/templates/live.html` | 13.865 | 43 | 1.378 |
| `internal/webui/templates/page.html` (prévia) | 12.951 | 61 | 904 |
| `internal/webui/templates/dialogs.html` | 5.460 | 8 | 1.271 |
| `internal/webui/templates/rewind.html` | 4.828 | 46 | 462 |
| `internal/webui/templates/media.html` | 4.553 | 41 | 388 |
| `internal/webui/templates/graphs.html` | 3.836 | 24 | 1.082 |
| `internal/adminui/templates/manage.html` | 14.330 | 10 | 2.261 |
| `internal/adminui/templates/admin.html` | 10.564 | 11 | 1.602 |
| `internal/adminui/templates/catalog-review.html` | 6.294 | 5 | 1.554 |
| `internal/adminui/templates/shell.html` | 6.250 | 10 | 1.543 |
| `internal/adminui/templates/settings.html` | 4.214 | 23 | 385 |
| `internal/adminui/templates/profile-review.html` | 1.871 | 1 | 1.870 |

- 36 blocos `{{define}}` em 12 arquivos. `page.html` (prévia) e `live.html` (páginas reais) repetem o `<head>` e a barra lateral; `shell.html` repete o `<head>` da administração.
- Os nomes dos blocos misturam convenções (`live-home`, `page.html`, `activity-grid`, `admin-head`).
- Diffs de template são ilegíveis: qualquer mudança altera uma linha de mais de mil caracteres.

### CSS

- `ui.css` (28 KB, 1.199 linhas): comentários marcam fases ("Roadmap 1", "Differences found in the first React/Go screenshot pass"), não componentes. Regras de um mesmo componente ficam espalhadas.
- `admin.css` (7 KB): praticamente uma linha.
- 18 media queries com 13 pontos de quebra diferentes: `40rem`, `48rem`, `60rem`, `64rem`, `80rem`, `96rem` e `480`, `639`, `640`, `650`, `767`, `900px`, além de `calc(40rem - 1px)`.
- Temas: `palettes.json` vira variáveis em `/web-assets/themes.css`, gerado pelo Go (`internal/webui/assets.go`). Isso funciona bem e deve ser mantido.

### Outros achados

- **Fontes externas.** `page.html`, `live.html` e `shell.html` carregam Jost, League Spartan e Inter do Google Fonts. Contraria o princípio de não depender de serviços externos em produção e vaza o IP do usuário para terceiros. `assets/Jost-Regular.ttf` e `assets/LeagueSpartan-Medium.ttf` existem, são copiados para a imagem Docker, mas nunca são servidos.
- **Cache.** Arquivos do `embed.FS` não têm data de modificação, então `/web-assets/ui.css` sai sem `Last-Modified`/`ETag` nem versão na URL.
- **Dependências de caminho** a ajustar numa mudança de lugar: `Makefile` (`node --check internal/webui/static/*.js`), `Dockerfile` (`COPY internal/` e `COPY assets/`), `scripts/validate_webui.py`, `scripts/capture_admin_webui.cjs`, `scripts/validate_admin_io_024.cjs`, `scripts/roadmap06_fixture.py`, `internal/i18n/catalog_ci_test.go`.
- `assets/` na raiz hoje guarda imagens do README (`icon.png`, `koito-original.svg`), capturas e as fontes não usadas.

## Onde os arquivos devem ficar

| Opção | Como | Prós | Contras |
| --- | --- | --- | --- |
| A. Manter em `internal/*/` | Só reformatar | Nenhuma mudança de caminho | Templates e CSS continuam escondidos em dois pacotes; não atende ao pedido |
| **B. Raiz, embutido** | `templates/` e `assets/` na raiz; um arquivo `web.go` (pacote `tunetrail`) na raiz com `//go:embed` | Lugar óbvio para quem edita front-end; continua um único executável; erro de template aparece nos testes, não em produção | Um arquivo Go na raiz; `Dockerfile` precisa copiar os dois diretórios na etapa de build |
| C. Raiz, externo (como `langs/`) | Lidos do disco: `TUNETRAIL_WEB_DIR`, depois ao lado do executável, depois a raiz do checkout | Editar CSS sem recompilar; usuário poderia customizar | Binário deixa de ser autossuficiente; versão do template pode divergir da do binário e quebrar em tempo de execução, porque os templates dependem das estruturas Go; mais um diretório a instalar e atualizar |
| **D. B + leitura do disco em desenvolvimento** | Igual a B; se `TUNETRAIL_WEB_DIR` estiver definida, lê do disco e recarrega a cada requisição | Edição ao vivo no `make ui.preview` e no `make api.dev` sem recompilar; produção igual a B | Um caminho de código a mais, só para desenvolvimento |

`go:embed` só alcança arquivos no diretório do pacote ou abaixo dele. Por isso, para usar `/templates` e `/assets` na raiz, a diretiva precisa ficar num pacote Go na raiz. O módulo já é `github.com/ebrgr/TuneTrail`; `internal/webui` e `internal/adminui` passariam a importar esse `embed.FS`.

**Recomendação: D.** Mantém o executável único, que é princípio do projeto, e dá a facilidade de editar sem recompilar durante o desenvolvimento.

### Árvore proposta

```text
templates/
  public/      base.html, home.html, artist.html, album.html, track.html,
               listens.html, rewind.html, profiles.html
  admin/       shell.html, admin.html, manage.html, settings.html,
               catalog-review.html, profile-review.html
  partials/    head.html, sidebar.html, cards.html, graphs.html,
               dialogs.html, avatar.html
assets/
  css/         ordem das camadas descrita abaixo
  js/          theme.js, ui.js, live.js, admin.js
  icons/       SVGs Lucide + LICENSE
  public/      favicon, manifest, ícones de app
  themes/      palettes.json
  fonts/       Jost, League Spartan, Inter (woff2) — fase opcional
  brand/       icon.png, icon.svg, koito-original.svg (README; não embutidos)
```

Cuidados:
- O `ParseFS` nomeia cada template pelo nome do arquivo, sem o diretório. Os nomes de arquivo precisam ser únicos entre `public/`, `admin/` e `partials/`.
- `go:embed` de diretório ignora arquivos que começam com `_` ou `.`. Não usar `_partial.html` como convenção.
- O `go:embed` lista só os subdiretórios de runtime (`assets/css assets/js assets/icons assets/public assets/themes`), para não embutir imagens do README nem capturas.
- As URLs públicas (`/web-assets/ui.css`, `/web-assets/icons/...`) continuam iguais na fase de mudança de lugar.

## Organização do CSS

Dividir por função, com `@layer` declarando a ordem:

```text
assets/css/
  00-layers.css     @layer reset, tokens, base, layout, components, pages, utilities;
  01-reset.css
  02-tokens.css     espaços, raios, sombras, tipografia, larguras; cores vêm de themes.css
  03-base.css       html, body, títulos, links, formulários
  04-layout.css     shell, sidebar, grades de página
  components/       card.css, rank.css, activity.css, dialog.css, forms.css, nav.css ...
  pages/            home.css, media.css, rewind.css, listens.css
  admin.css         reformatado, mesma divisão quando crescer
  99-utilities.css  poucos utilitários próprios (visually-hidden, stack, cluster)
```

Como servir sem etapa de build: o Go concatena os arquivos em ordem alfabética na inicialização e serve um único `/web-assets/ui.css`, com hash do conteúdo para cache longo (`ui.css?v=<hash>` ou `ui.<hash>.css`). São poucas linhas de Go, sem ferramenta nova. Alternativa descartada: vários `<link>` ou `@import`, que aumentam o número de requisições.

Pontos de quebra: reduzir os 13 a três (`40rem`, `64rem`, `96rem`) e usar container queries nos cards e grades. Variáveis CSS não funcionam dentro de `@media`, então a escala é garantida por uma verificação automática (abaixo).

## Organização dos templates

- Uma tag por linha nos blocos estruturais, indentação de 2 espaços, atributos longos quebrados em várias linhas.
- Um arquivo por página e parciais compartilhadas: um único `head.html` para prévia, páginas reais e administração.
- Nomes de bloco uniformes: `page/home`, `partial/sidebar`, `admin/settings`.
- Usar `{{-` e `-}}` onde a quebra de linha mudaria o espaço entre elementos em linha.
- Verificação automática em `make ui.check` (`scripts/check_web_style.py`): nenhuma linha acima de 160 caracteres em `templates/` e `assets/css/`, e `@media` só com os pontos de quebra permitidos. Impede a volta das linhas únicas.

## Como verificar cada fase

1. **Antes de tudo:** um teste que renderiza todas as páginas da prévia e da administração com dados fictícios e grava o HTML como referência (`backlog/evidencias/reorganizacao-web/`).
2. **Mudança de lugar:** HTML renderizado e CSS servido **idênticos byte a byte** à referência.
3. **Reformatação dos templates:** HTML idêntico depois de normalizar espaços em branco, mais capturas em 320, 390, 768 e 1440 px nos temas Yuu e Pearl com os scripts de captura existentes.
4. **Divisão do CSS:** o CSS concatenado é igual ao `ui.css` atual, salvo espaços e a ordem documentada.
5. **Tokens e pontos de quebra:** muda o visual de propósito. Capturas antes/depois e aceite visual do usuário.

## Fases propostas (cada uma vira uma tarefa)

| Fase | Escopo | Visível ao usuário? |
| --- | --- | --- |
| 1 | Mover para `templates/` e `assets/` + `web.go` na raiz + `TUNETRAIL_WEB_DIR` para desenvolvimento; ajustar Dockerfile, Makefile e scripts | Não (HTML idêntico) |
| 2 | Reformatar templates e unificar o `<head>`/barra lateral | Não (HTML equivalente) |
| 3 | Dividir `ui.css`/`admin.css` em camadas, concatenação no Go, hash para cache | Não (CSS equivalente) |
| 4 | Tokens e três pontos de quebra, container queries | Sim (layout responsivo) |
| 5 | Fontes servidas pelo próprio TuneTrail, sem Google Fonts | Sim (privacidade, funciona offline) |

As fases 1–3 não mudam o comportamento e podem ser verificadas por comparação. As fases 4 e 5 precisam de aceite visual.

## Decisões para o usuário

1. Embutido com leitura do disco em desenvolvimento (D, recomendado) ou externo como `langs/` (C).
2. Nomes `templates/` e `assets/` na raiz, movendo as imagens do README para `assets/brand/`.
3. Incluir a fase 5 (fontes locais) neste plano ou tratá-la como tarefa separada.
