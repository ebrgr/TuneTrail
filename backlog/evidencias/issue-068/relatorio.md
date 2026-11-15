# ISSUE-068 — Verificação da divisão do CSS em camadas

Data: 2026-11-15. Base de comparação: `main` em `cfd6717`.

## Estrutura

`ui.css` servido = concatenação, na ordem do `assets/manifest.json`, de 18 partes em `assets/css/ui/`:

| Camada | Arquivos |
| --- | --- |
| reset | `reset.css` (declara `@layer reset, tokens, base, layout, components, pages, utilities;`) |
| tokens | `tokens.css` |
| base | `base.css` |
| layout | `layout/shell.css` |
| components | `components/{profile,section,media-list,activity,dialog,theme-panel,search,forms}.css` |
| pages | `pages/{home,profiles,listing,media,rewind}.css` |
| utilities | `utilities.css` (`[hidden]`, `.sr-only`) |

`admin.css` passou para `assets/css/admin/admin.css`, dentro de `@layer pages`. O maior arquivo tem 323 linhas (`admin/admin.css`); nenhum passa de 400.

A divisão foi gerada por `ferramentas/split.py` a partir de `git show main:assets/css/ui.css`. Cada regra vai para um arquivo segundo o seletor, e cada arquivo mantém a ordem original das suas regras. Media queries com regras de vários componentes foram repartidas, cada pedaço com a mesma condição.

## Ordem documentada (critério #1)

- `ferramentas/verify.py`: o bundle contém exatamente as 366 regras originais (mesma condição de media, seletor e declarações, ignorando espaços e comentários). Nenhuma regra foi removida ou alterada. Nenhuma regra morta foi removida nesta tarefa: sem uma prova de que não é usada, remover mudaria o conjunto verificado.
- Mudança de ordem: regras passaram a ficar agrupadas por função. Com `@layer`, uma camada posterior ganha independente da especificidade; por isso três regras foram colocadas na camada em que competiam antes:
  - `.nav-button .avatar` e `.nav-button .avatar svg` em `components/profile.css` (junto de `.avatar`); em `layout` o avatar da navegação passava de 18 px para 32 px.
  - `.old-week` em `components/activity.css`; em `media-list.css` ficava antes de `.activity-slot` e voltava a aparecer em telas estreitas.
  - `h2.search-section-title` em `components/search.css`; em `base` perdia para `dialog h2`.
  - O `:focus-visible` genérico da administração ficou num bloco `@layer base` no fim de `admin/admin.css`, onde continua perdendo por especificidade para `:is(a,button,input,select,textarea):focus-visible`.
- `ferramentas/conflicts.py` lista pares de regras cujo vencedor poderia mudar (entre camadas, ou dentro da mesma camada com ordem invertida). Os candidatos restantes foram revistos: elementos que não coincidem nos templates, mesmo valor (`padding-right: 2.5rem` do `select`) ou `!important`.

## Estilos computados

`scripts/compare_computed_styles.cjs` (nova ferramenta) compara `getComputedStyle` de todos os elementos e de `::before`/`::after` em 11 páginas × 2 temas × 5 larguras (360, 768, 1024, 1440, 1600), com fontes externas bloqueadas.

- Primeira rodada: 9.046 diferenças (as quatro causas acima).
- Após as correções: **0 diferenças em 70.710 elementos, 110 visões** (`estilos-computados.json`).
- Não cobre hover, foco nem diálogos abertos; esses estados foram tratados pela análise estática acima.

## Capturas (critério #4)

`scripts/capture_ui_base.cjs` em `main` e na branch: 132 capturas, sem overflow e sem falha de fonte. 126 são idênticas byte a byte. As 6 restantes (administração, 768 e 1024 px) diferem em 19 a 25 pixels num quadrado de 13×13 px (colunas 20–32, linhas 839–851), o ícone de configurações no pé da barra lateral. Em 1024 px duas capturas consecutivas da própria `main` diferem nos mesmos pixels: é instabilidade de renderização desse ícone, não efeito do CSS, o que é coerente com os estilos computados idênticos.

## URL com hash (critério #2)

O hash e o `Cache-Control: immutable` existem desde a ISSUE-106. Os testes novos em `internal/webassets/bundle_test.go` provam que o bundle é a concatenação das partes na ordem do manifesto, que alterar uma parte muda a URL de `ui.css` sem mudar a de `admin.css`, e que listas só são aceitas para folhas de estilo públicas, não vazias e sem duplicatas.

## Outras verificações

- `check_web_style.py` agora percorre `assets/css/` recursivamente e recusa partes com mais de 400 linhas (testes em `test_check_web_style.py`).
- Referência `backlog/evidencias/issue-067/reference` regenerada: só mudam `ui.css`, `admin.css`, os hashes `?v=` e `data-ui-generation` nas páginas.
- `make check`: verde, incluindo 12/12 jornadas E2E.

## Reviewer

Parecer em `parecer-reviewer.md`, sem itens bloqueantes. Os cinco ajustes foram aplicados:

1. `url()` em parte aninhada: documentado no `scripts/README.md` (resolvido a partir do `ui.css` servido) e coberto por `TestNestedPartURLResolvesFromServedStylesheet`; um caminho que sai da raiz pública falha ao carregar.
2. `admin/admin.css` repete a declaração de ordem das camadas no topo.
3. `bundle_test.go`: o teste de concatenação usa partes próprias (uma sem `\n` final, outra aninhada) e o da declaração de ordem confirma que ela é a primeira regra `@layer`.
4. `scripts/README.md` cita a exceção do `:focus-visible` na `base`.
5. As ferramentas em `ferramentas/` localizam o repositório pelo próprio caminho e leem a base fixa `cfd6717`; `split.py` reproduz os arquivos de forma idêntica.
