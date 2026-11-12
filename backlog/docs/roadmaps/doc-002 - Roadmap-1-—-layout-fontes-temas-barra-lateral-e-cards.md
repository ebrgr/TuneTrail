---
id: doc-002
title: Roadmap 1 — layout, fontes, temas, barra lateral e cards
type: specification
created_date: '2026-11-12T23:39:00Z'
updated_date: '2026-11-12T23:39:00Z'
tags:
  - roadmap
---
> Importado de [docs-dev/ROADMAP_01_INTERFACE.md](../../../docs-dev/ROADMAP_01_INTERFACE.md).

# Roadmap 1 — layout, fontes, temas, barra lateral e cards

Análise em 4 de novembro de 2026, sobre os arquivos locais; HEAD de referência:
`72e4be2`. **Status atualizado: primeira prévia Go implementada e validada.**
Veja [implementação, screenshots e limites](../../../docs-dev/comparisons/roadmap-01/README.md).
O restante deste documento preserva a análise e a proposta originais. Na fase
de análise não houve inicialização da aplicação, consulta ao banco nem validação
no navegador; a prévia foi implementada posteriormente, com fixtures isoladas.

O banco vigente é **SQLite com esquema consolidado v3**; veja
[BANCO_DE_DADOS.md](../../../docs-dev/BANCO_DE_DADOS.md). Este passo trata da apresentação e não
requer mudança de esquema.

## Objetivo e limite deste passo

Reproduzir a identidade visual atual com `html/template`, CSS convencional e
JavaScript pequeno para comportamentos locais. Preservar a aparência do fork SUM,
incluindo cabeçalho e seletor de perfis, em vez de redesenhar o produto.

A entrega proposta é uma composição visual executável com dados fictícios e
determinísticos: estrutura de página, tipografia, todos os temas, navegação,
avatares e famílias de cards. A migração das consultas e das páginas reais fica
no passo 2; administração completa, busca funcional e mutações ficam nos passos
correspondentes. A prévia deve identificar esses limites, sem botões que aparentem
salvar dados reais ou links que levem acidentalmente à administração em uso.

React permanece disponível durante a comparação. Removê-lo, retirar Node do
build e substituir as rotas públicas são decisões posteriores à paridade visual.

## Mapa do código atual

Caminhos relativos à raiz do repositório:

| Elemento | Fonte atual | Reprodução proposta |
| --- | --- | --- |
| Documento HTML, metadados, fontes e ícones da aplicação | `client/app/root.tsx` | Template base e arquivos estáticos |
| Estrutura compartilhada e rodapé | `client/app/components/AppShell.tsx`, `Footer.tsx` | Parciais de estrutura e rodapé |
| Cabeçalho do perfil | `client/app/routes/ProfileLayout.tsx` | Parcial com identidade pública explícita |
| Composição da home e seleção de perfis | `client/app/routes/Home.tsx`, `Profiles.tsx` | Composições de referência, inicialmente com fixtures |
| Estilos globais e utilitários de cores | `client/app/app.css`, `themes.css` | CSS base, componentes e regras responsivas |
| Paletas e nomes das variáveis | `client/app/styles/themes.css.ts`, `vars.css.ts` | CSS estático por `data-theme` |
| Persistência e seleção de tema | `client/app/providers/ThemeProvider.tsx`, `components/themeSwitcher/` | Script local de tema e formulário de aparência |
| Navegação e seletor de perfil | `client/app/components/sidebar/` | Navegação HTML, SVGs e diálogo |
| Aparência do avatar | `client/app/components/ProfileAvatar.tsx` | Parcial de avatar e fallback |
| Cards com destaque | `client/app/components/TopArtistsCard.tsx`, `TopAlbumsCard.tsx` | Parcial compartilhada com variantes |
| Linhas de mídia, imagens e títulos | `client/app/components/primitives/`, `ArtistLinks.tsx` | Parciais reutilizáveis |
| Ranking e histórico recente | `client/app/components/TopItemList.tsx`, `TopTracks.tsx`, `LastPlayed.tsx`, `ListensTable.tsx` | Listas/tabelas, sem transformá-las em cards com borda |
| Estatísticas, atividade e período | `client/app/components/AllTimeStats.tsx`, `ActivityGrid.tsx`, `PeriodSelector.tsx` | Estrutura visual com dados de demonstração |
| Diálogos e dicas | `client/app/components/modals/Modal.tsx`, `Popup.tsx`, `SettingsModal.tsx` | Diálogo, dicas CSS e controles locais |

`assets/screenshot1_2.png` foi inspecionada como referência de composição. Ela não
mostra o cabeçalho/seletor SUM e não substitui capturas atuais. Os caminhos
`docs/screenshots/multiuser/` mencionados no documento de PR não existem nesta
cópia local. Será necessário produzir uma referência atual antes de aceitar a
fidelidade da implementação.

## Layout e medidas a preservar

As medidas em pixels abaixo assumem a base usual de 16 px por rem. Os breakpoints
Tailwind foram conferidos em `client/node_modules/tailwindcss/theme.css`; a nova
interface deverá escrevê-los explicitamente em CSS, sem depender de Tailwind.

| Regra | Comportamento observado |
| --- | --- |
| `sm`: 40 rem / 640 px | Barra horizontal vira lateral vertical fixa; aparece o recuo de 58 px do conteúdo |
| `md`: 48 rem / 768 px | Espaçamento vertical da home passa de 40 para 48 px; intervalo do bloco superior passa de 40 para 80 px |
| Tipografia: 60 rem / 960 px | Títulos grandes mudam de tamanho; breakpoint próprio, diferente de `lg` |
| `lg`: 64 rem / 1024 px | Estatísticas e atividade ficam lado a lado; histórico admite largura máxima de 1200 px |
| `xl`: 80 rem / 1280 px | Disponível nas regras Tailwind; conferir usos em cada página ao migrá-la |
| `2xl`: 96 rem / 1536 px | A largura máxima de Top tracks na home muda de 750 para 450 px |

A estrutura usa altura mínima de tela e uma coluna flexível com rodapé. A home
usa flex com quebra de linha e intervalo de 40 px entre blocos, e não uma grade
fixa de três colunas. As margens laterais são 16 px no celular e 40 px a partir
de `sm`. O cabeçalho do perfil tem borda inferior, avatar de 32 px, nome e slug.

A seleção de perfis tem largura máxima de 56 rem e passa de uma coluna para duas
em `sm` e três em `lg`. Cada card tem 24 px de padding e avatar de 56 px.

**Ponto de atenção:** existem mínimos de 350 px em blocos da home. Somados às
margens, podem causar transbordamento em telas estreitas; isso é uma hipótese
de leitura, ainda sem medição no navegador. Usar a largura disponível como limite
na proposta e comparar em 360/390/400 px, registrando qualquer diferença deliberada.
Não reproduzir um eventual overflow como requisito de fidelidade.

## Fontes e hierarquia

- Texto: Jost, seguido de Inter e fontes do sistema. Jost e League Spartan são
  carregadas por Google Fonts em `app.css`; Inter tem um carregamento separado
  em `root.tsx`.
- Títulos e `.header-font`: League Spartan. `h1` e `h2` usam peso 600; `h3`
  usa peso 400 e caixa alta; `h4` usa peso 600.
- `h1`: 36 px, passando a 78 px em 60 rem. `h2`: 28 → 36 px. `h3`: 18 px;
  `h4`: 16 px. O `h1` usa linha de 3 rem antes do breakpoint e 1,3 em depois.
- Linhas de mídia e tabelas combinam tamanhos de 12/14 e 13/15 px conforme o
  componente e a largura. Títulos dos destaques usam 30 px e peso 600.

Há `assets/Jost-Regular.ttf` e `assets/LeagueSpartan-Medium.ttf`, usados pela
geração de imagens de retrospectiva em Go. Não equivalem automaticamente às
fontes web variáveis, seus pesos e itálicos. Não substituir uma pela outra
presumindo fidelidade.

Proposta: manter inicialmente o carregamento web equivalente para estabelecer
a referência; depois servir fontes locais equivalentes, com pesos, itálicos e
licenças próprios verificados. Essa segunda mudança deve ser comparada à referência.
Preservar também line-height, normalização de margens e box-sizing: retirar
Tailwind sem substituir seu reset altera a aparência mesmo com as fontes certas.

## Temas

Existem **12 temas**: `yuu`, `varia`, `midnight`, `catppuccin`, `autumn`, `black`,
`wine`, `pearl`, `rosebud`, `urim`, `match` e `lemon`.

Cada paleta contém 14 valores: três fundos, três cores de texto, principal e
principal atenuada, destaque e destaque atenuado, erro, aviso, informação e
sucesso. `--color-border` referencia `--color-bg-tertiary`.

Copiar os valores exatos para seletores `[data-theme="..."]`, mantendo os nomes
das variáveis existentes. Vanilla Extract apenas gera essas regras; não é
necessário em tempo de execução na reprodução. Não reduzir o conjunto a um tema
escuro e um claro: `pearl` também precisa de comparação própria.

O padrão vem da configuração `KOITO_DEFAULT_THEME`, com fallback `yuu` no cliente.
A preferência é do navegador, compartilhada entre perfis, e utiliza as chaves
`theme`, `custom-theme` e `bgcolor` no localStorage. Trocar perfil não deve trocar
tema nem gravar uma preferência em dados administrativos.

Proposta para as páginas Go: renderizar um tema padrão válido no HTML e aplicar
a preferência local por um pequeno script no início do documento, antes da
primeira pintura. A leitura do storage deve tolerar indisponibilidade e JSON
inválido; o tema personalizado deve aceitar apenas as 14 propriedades de cor
previstas, sem inserir CSS ou HTML arbitrários. Não é necessária tabela no banco
nem cookie de perfil para isso.

**Possível defeito observado, sem reprodução:** o efeito de `ThemeProvider.tsx`
acessa `themes[themeName].bg` antes de tratar `themeName === "custom"`, mas não
há paleta `custom` no mapa. A nova implementação deve obter o fundo da paleta
efetivamente resolvida. Isso não autoriza uma correção funcional da SPA nesta
etapa documental; tratar a correção separadamente se for executada.

## Barra lateral e interações

Preservar a ordem e as condições de exibição atuais:

1. Seletor de perfis; busca somente quando há perfil ativo.
2. Home; retrospectiva somente quando há perfil ativo.
3. About externo e Settings no grupo final.

Ícones principais têm 18 px; os botões têm 8 px de padding. No desktop, a barra
tem padding horizontal de 12 px e vertical de 40 px, borda direita e altura da
tela. No celular, tem borda inferior e ocupa a largura da página, sem posição
fixa no componente atual. A largura de 58 px é o recuo declarado no AppShell;
a largura efetiva da barra é automática e deve ser medida na referência.

Ícones vêm de Lucide e de componentes SVG próprios. Usar SVGs estáticos
equivalentes, preservando geometria, traço, tamanho e atribuições de licença;
não substituir por emojis ou por outra família visual.

O seletor usa diálogo de até 480 px, lista com avatares de 40 px, indicação de
seleção e link para todos os perfis. O avatar da barra tem 18 px. Preservar a
ordem foto → ícone permitido → iniciais; erro de imagem também deve cair no
fallback. As iniciais usam as primeiras letras de até duas palavras, incluindo
nomes com acentos.

Settings abre um diálogo de até 1000 px, altura solicitada de 700 px limitada
a 80% da tela. As abas ficam em coluna lateral a partir de `sm` e acima do
conteúdo no celular. A administração permanece dentro de Settings; não adicionar
um ícone administrativo separado. Neste passo, reproduzir Aparência e a estrutura
das demais seções, deixando a integração administrativa para o passo 4.

JavaScript local proposto: tema, abertura/fechamento de diálogos, abas,
atalhos `/` e `\` fora de campos de edição e fallback de imagem. Avaliar
`<dialog>` nativo para preservar Escape, foco contido e retorno de foco, com
estilos explícitos e fechamento por clique fora. Dicas devem aparecer também
por foco de teclado; as dicas atuais são exibidas por hover apenas no desktop.
Animações atuais duram 100 ms; fundo muda em 300 ms. Prever movimento reduzido.

Links finais devem usar o slug explícito e preservar tela/período ao trocar de
perfil conforme `client/api/profile-paths.ts`. O perfil não será inferido de um
cookie global. Busca, administração e retrospectiva reais não fazem parte da
prévia visual.

## Cards e blocos reutilizáveis

| Família | Características a preservar |
| --- | --- |
| Superfície `.card` | Fundo secundário, borda de 1 px no fundo terciário e raio de 8 px |
| Título de seção | League Spartan, caixa alta, texto secundário, recuo opcional de 24 px, seta no hover do link |
| Destaque de artista/álbum | Largura de referência de 350 px; imagem quadrada; gradiente com os mesmos pontos de transição; título sobreposto; até quatro linhas menores após o primeiro colocado |
| Linha de mídia | Capas de 56, 90 ou 125 px; intervalo de 12 px; título em até duas linhas; subtítulo e metadados opcionais |
| Ranking de faixas | Posição com dois dígitos, primeira em cor principal, contagem alinhada à direita e separadores; sem caixa `.card` externa |
| Estatísticas | Duas colunas, números destacados, sem superfície de card externa |
| Histórico recente | Tabela, miniaturas de 32 px, tempo à direita e linha de tocando agora; sem caixa `.card` externa |
| Seletor de período | Superfície `.card`, opções em caixa alta e estado ativo; integração de filtro no passo 2 |
| Atividade | Superfície `.card`, sete linhas, legenda e sequência; células de 9/10 px e espaços de 2/3 px |
| Perfil | Nome e slug, avatar circular e hover; estado sem perfis também reproduzido |

Nem todo bloco é um card com borda. Envolver rankings e estatísticas na mesma
caixa mudaria a composição. No destaque, copiar o gradiente atual de
`TopArtistsCard.tsx`/`TopAlbumsCard.tsx`, em vez de aproximá-lo com uma faixa escura
simples. Preservar `srcset`, `sizes`, dimensões e carregamento adiado onde aplicável.

Preparar exemplos de conteúdo normal, vazio, erro, nomes longos, Unicode e imagem
indisponível. Skeletons atuais são estados de consultas no navegador; páginas
renderizadas em Go podem entregar conteúdo pronto. Só usar skeleton quando houver
carregamento parcial real, sem introduzir uma espera artificial.

A atividade atual mostra 36 semanas e remove dez quando `window.innerWidth <= 640`.
Isso não coincide exatamente com o início do `sm` (640 px). A futura versão pode
gerar células no servidor e ocultar colunas por CSS, mas deve registrar a regra
adotada e comparar em 639/640/641 px. A lógica real de calendário, início da semana
e fuso pertence ao passo 2 e deve respeitar `KOITO_FORCE_TZ`.

## Organização proposta e convivência com React

Adicionar uma camada própria, por exemplo `internal/webui/`, contendo renderização,
modelos de apresentação, `templates/` e `static/` (CSS, SVG e JavaScript). Templates
podem ser incorporados com `go:embed` e usar `html/template` com escape automático.
Os modelos recebem texto, links e imagens; não recebem componentes React nem HTML
arbitrário de catálogo. A decomposição proposta é base → estrutura → parciais de
navegação, perfil, título, imagem, linha de mídia, destaque e estados.

Para o passo 1, usar fixtures públicas/fictícias em um servidor de prévia isolado
em loopback, com portas distintas das habituais e sem abrir o banco de uso. Não
adicionar rotas de demonstração à produção. Isso permite revisar o resultado
antes de alterar o roteamento real.

O ponto de integração posterior é `engine/routes.go`, onde `fileServer` hoje serve
a SPA com fallback para `index.html`. Não basta acrescentar uma rota curinga HTML:
é necessário definir quais páginas já migraram e preservar as demais na SPA,
além das rotas `/apis`, `/image`, arquivos públicos e slugs reservados. Assets novos
devem ter namespace próprio e URLs versionadas se usarem cache `immutable`.

Handlers HTML futuros deverão chamar os serviços e consultas Go compartilhados,
sem fazer HTTP à própria API e sem chamar handlers JSON com um response recorder.
O middleware atual de perfil responde erros JSON; compartilhar a resolução/escopo
não significa reutilizar sua resposta de erro em páginas HTML. Preservar
`db.WithProfile` e a separação de sessão administrativa ao integrar dados reais.
As mudanças nos handlers e no roteamento ficam fora desta análise visual.

## Sequência de execução proposta

1. Registrar issue da reprodução antes de escrever código funcional, conforme
   `agents.md`. Capturar a interface React atual com dados descartáveis e fixos.
2. Criar estrutura de prévia isolada e normalização CSS; reproduzir fontes,
   variáveis, 12 paletas e preferência local, incluindo tema personalizado.
3. Reproduzir estrutura de página, cabeçalho de perfil, barra, rodapé e avatares.
4. Reproduzir as famílias de cards/listas e estados com os mesmos dados de referência.
5. Acrescentar diálogos, Aparência, dicas, teclado e fallbacks necessários à prévia.
6. Comparar desktop/celular e registrar diferenças, capturas e validações neste
   diretório. Só depois avançar para páginas e dados reais no passo 2.

## Critérios para aceitar a implementação

- Comparações lado a lado usando os mesmos dados, fontes carregadas, tema,
  período, viewport, escala e navegador; não comparar conteúdo diferente.
- Desktop em 1280 e 1536 px; celular em 360, 390 e 400 px; conferir transições
  em 639/640/641, 767/768, 959/960 e 1023/1024 px. Se necessário, expandir apenas
  onde uma diferença aparecer.
- Todos os 12 temas verificados; comparação detalhada de `yuu`, `black` e `pearl`,
  incluindo bordas, gradientes, foco, hover, nomes longos e ausência de imagens.
- Preferência de tema sobrevive a recarga, reset e troca de perfil; storage
  indisponível, tema desconhecido e JSON inválido não impedem a página de abrir.
- Links e botões com nomes acessíveis; teclado abre/fecha diálogos, percorre abas,
  retorna ao acionador e não dispara atalhos durante digitação.
- Sem rolagem horizontal involuntária nos tamanhos móveis; imagens mantêm espaço
  reservado e textos não encobrem controles.
- Prévia executada em Go com CSS/JS estáticos, sem Node em execução, sem consultar
  o banco real nem alterar ingestão, API ou esquema v3. A SPA continua disponível.

Para a implementação, testes automatizados devem cobrir riscos concretos como
escape de texto, validação de paleta e preservação do contexto de perfil quando
integrado. A fidelidade visual exige inspeção no navegador; um build aprovado
sozinho não comprova paridade. Ainda não há evidência de paridade ou de ganho de
desempenho nesta etapa.

## Registro desta análise (histórico)

Inspecionados os arquivos indicados, os breakpoints instalados e uma captura
existente. Produzido este plano e vinculado ao roadmap. Foram alterados somente
documentos; nenhuma issue foi criada, nenhum código funcional foi escrito e
nenhum teste da aplicação foi executado nesta análise. A implementação posterior
e suas validações constam no [registro de comparação](../../../docs-dev/comparisons/roadmap-01/README.md).
