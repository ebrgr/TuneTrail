# Issue 023 — páginas públicas, mídia e retrospectiva

Execução em 05/11/2026. [Plano](<../../docs/roadmaps/doc-009 - Roadmap-6-—-comparação-visual-e-decisão-sobre-React.md>) ·
[matriz](ISSUE_022_MATRIZ.md) · [manifesto de capturas](screenshots-023.json).
SQLite v3 fictício e cópias separadas; Chrome 156.0.8078.80, Node 24.15.0,
fuso `America/Sao_Paulo`, viewports 1440×1000 e 390×844. A árvore inclui alterações
locais não commitadas; o commit de referência está no manifesto. Os dados da
fixture têm relógio fixo, mas o relógio do servidor é real.

## Correção

Sem JavaScript, escolha de perfil e busca na barra lateral Go eram botões para
abrir dialogs que não funcionavam. Regressão falhou antes da correção. O template
agora oferece links HTML para a seleção e busca pessoal e oculta os botões que
exigem scripts. Em navegador sem JS: Alpha → seletor → Beta → busca pessoal
funcionou. A escolha de tema continua exigindo JS e uma mensagem informa isso.
Nenhum JavaScript foi acrescentado à aplicação.

## Jornadas e verificações

- Go e React hidratado: seleção, troca Alpha→Beta, recarga, voltar/avançar, duas
  abas, busca de item pessoal, acesso à faixa e retrospectiva; links de ranking e
  histórico e filtros ano 2018/mês 11 produziram dois registros. As diferentes
  interações de busca (documento Go e modal React) foram exercitadas.
- Seletores com zero/um perfil passaram nos dois modos; perfil inexistente foi
  capturado com a mensagem própria de cada interface. Home de perfil vazio,
  imagem ausente, título longo e retrospectivas mensal e anual foram capturados.
- Histórico de 1009 escutas: segunda página continha 100 linhas em ambos os modos,
  desktop e celular. O controle de tempo da retrospectiva mensal foi exercitado
  em ambos; Go também com JS desativado.
- `validate_graphs_023.py`: 192 intervalos de API e 96 da tabela Go, para
  artista/álbum/faixa e Alpha/Beta, confrontados com instantes literais da fixture
  e divisões calculadas independentemente. O fim do intervalo usa o relógio real
  informado pela resposta; HTML e API podem ser consultados em segundos diferentes.
  Atividade de novembro/2018 em São Paulo teve 1 escuta em cada dia 3 e 4; em UTC,
  as duas ocorreram no dia 4, confirmado em processos Go/React separados.
- Amostra de 52 combinações de largura e rota: Go sem transbordamento horizontal
  **do documento** nas duas páginas verificadas de 320 a 1440 px. React apresentou
  largura 342/320 na home e 389/320 na faixa, e 362/360 na home. A verificação não
  garante ausência de cortes internos nem cobre todas as páginas.
- Nas nove paletas além de Yuu/Pearl/Black, os nomes aplicados foram conferidos em
  ambos os modos. Isso não é auditoria de contraste ou legibilidade de cada tela.
- `go test ./...`, `go test -race ./internal/webui ./internal/webquery
  ./internal/listening`, `go vet ./internal/webui/...` e regressão no-JS passaram.
  Algumas suítes usaram cache. As cópias SQLite usadas passaram `integrity_check`
  e `foreign_key_check`; servidores temporários foram encerrados.

## Comparação visual

[120 capturas pareadas e de regiões intermediárias](screenshots-023.json), com
arquivos na pasta [screenshots-023](screenshots-023). Os pares principais cobrem
Yuu, Pearl e Black; outros estados usam Yuu. As nove paletas restantes tiveram
apenas verificação de aplicação. Capturas adicionais incluem 0/1 perfis, erro,
capa ausente, nome longo, página 2 e retrospectiva anual.

| Exemplo | Go | React |
| --- | --- | --- |
| Home celular Yuu | [Go](screenshots-023/go-home-yuu-mobile.jpg) | [React](screenshots-023/react-home-yuu-mobile.jpg) |
| Faixa desktop Yuu | [Go](screenshots-023/go-track-yuu-desktop.jpg) | [React](screenshots-023/react-track-yuu-desktop.jpg) |
| Retrospectiva desktop Yuu | [Go](screenshots-023/go-rewind-yuu-desktop.jpg) | [React](screenshots-023/react-rewind-yuu-desktop.jpg) |
| Perfil vazio celular Black | [Go](screenshots-023/go-empty-black-mobile.jpg) | [React](screenshots-023/react-empty-black-mobile.jpg) |
| Histórico página 2 celular | [Go](screenshots-023/go-history-large-yuu-mobile.jpg) | [React](screenshots-023/react-history-large-yuu-mobile.jpg) |

Diferenças observadas, encaminhadas para avaliação conjunta na 026: React usa
modal para busca e Go uma página; gráficos Go incluem tabela sem JS; controles de
retrospectiva e apresentação do cabeçalho/gradiente diferem; datas são formatadas
de forma diferente no navegador. As fontes remotas foram bloqueadas: React e Go
caíram em famílias de fallback diferentes. **Paridade tipográfica permanece
inconclusiva**, e essas imagens não aprovam diferenças como decisão de produto.
O transbordamento React em 320/360 px é resultado observado na referência, não
foi reproduzido como defeito Go nas páginas testadas.

## Limites

Não houve auditoria completa de acessibilidade/contraste, todas as combinações de
atores, filtros ou breakpoints por tela, nem inspeção de todos os 120 pares em
cada estado; 024/026 consolidam essa cobertura e sua aceitação. Sem JS, aparência
não pode ser alterada. API e tabela Go de interesse foram comparadas por intervalo;
o gráfico React foi comparado visualmente, não por extração numérica de cada ponto.
As imagens foram capturadas com JS; o teste de navegação Go sem JS foi separado.
As diferenças visuais não foram aceitas pelo usuário nesta issue; React continua
padrão e sua retirada depende da decisão posterior da etapa 6.
