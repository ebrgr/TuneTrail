# Issue 009 — integração, gestão e comparação visual

**Situação atual: issue encerrada por aceite do usuário em 05/11/2026.**
O restante preserva o histórico de implementação e verificação.

A integração está implementada no código: em `KOITO_WEB_UI=go`, artista, álbum,
faixa e retrospectiva usam as páginas Go; seleção, home, rankings e histórico
continuam na mesma interface. `KOITO_WEB_UI=react` continua sendo o padrão. As
rotas `/{slug}/manage/{artist|album|track}/{id}` usam os editores React existentes,
com login administrativo e retorno à página pública. A SPA e seu build continuam
necessários. Nenhuma instância existente foi reiniciada e não há nova migration.

O arraste de capa recebe explicitamente o tipo e o ID da imagem: em uma faixa,
a imagem pertence ao álbum da faixa. Os ouvintes do navegador são atualizados se
o item ou o perfil muda. Uma mesclagem inversa navega para o item sobrevivente;
a exclusão retorna à home do perfil. O catálogo permanece compartilhado, enquanto
escutas e estatísticas permanecem restritas ao perfil.

## Validações

- `go test ./...`, `make ui.check`, `npm run typecheck`, `npm test` e
  `npm run build`: passaram.
- `scripts/validate_webui.py`: passou nos dois modos com cópias SQLite v3,
  incluindo rotas Go/React, isolamento, IDs inválidos, interest, autorização e
  integridade. [Resultado HTTP](validation-009.json).
- `scripts/validate_multiuser.py`: passou nas bases nova e migrada v2, incluindo
  ingestão concorrente, administração, importação/exportação e reinício.
- No navegador, com conta e base descartáveis: navegação de artista a faixa,
  álbum, histórico e retrospectiva; voltar/avançar; dois perfis em abas separadas;
  entrada em gestão, login, criação de alias e retorno Go; adição de escuta com
  atualização da contagem; mesclagem de faixa nas duas direções, incluindo retorno
  ao item sobrevivente. A exclusão de uma faixa foi testada pela API: 401 sem
  sessão, 403 com origem inválida, 204 com sessão e 404 na leitura posterior.
- O Advisor (`@Sol`, esforço leve) revisou a integração. Identificou confusão entre
  o slug `manage` e o segmento de rota de gestão; o caso foi corrigido, ganhou
  regressão no cliente e recebeu parecer final **aprovado**.

## Comparações registradas

As capturas de página inteira usam perfis fictícios e cópias equivalentes da base
SQLite v3, `America/Sao_Paulo`, viewport desktop 1365 × 900 e celular 390 × 844.
O [manifesto](screenshots.json) lista rota, tema e arquivo de cada captura. Também
há imagens com Pearl e com os minutos ativados; a gestão foi capturada em sessão
de teste. A base não é o banco em uso.

| Página | React desktop | Go desktop | React celular | Go celular |
| --- | --- | --- | --- | --- |
| Artista | [React](screenshots/react-artist-desktop.jpg) | [Go](screenshots/go-artist-desktop.jpg) | [React](screenshots/react-artist-mobile.jpg) | [Go](screenshots/go-artist-mobile.jpg) |
| Álbum | [React](screenshots/react-album-desktop.jpg) | [Go](screenshots/go-album-desktop.jpg) | [React](screenshots/react-album-mobile.jpg) | [Go](screenshots/go-album-mobile.jpg) |
| Faixa | [React](screenshots/react-track-desktop.jpg) | [Go](screenshots/go-track-desktop.jpg) | [React](screenshots/react-track-mobile.jpg) | [Go](screenshots/go-track-mobile.jpg) |
| Retrospectiva mensal | [React](screenshots/react-rewind-month-desktop.jpg) | [Go](screenshots/go-rewind-month-desktop.jpg) | [React](screenshots/react-rewind-month-mobile.jpg) | [Go](screenshots/go-rewind-month-mobile.jpg) |
| Retrospectiva anual | [React](screenshots/react-rewind-year-desktop.jpg) | [Go](screenshots/go-rewind-year-desktop.jpg) | [React](screenshots/react-rewind-year-mobile.jpg) | [Go](screenshots/go-rewind-year-mobile.jpg) |
| Perfil vazio | [React](screenshots/react-empty-desktop.jpg) | [Go](screenshots/go-empty-desktop.jpg) | [React](screenshots/react-empty-mobile.jpg) | [Go](screenshots/go-empty-mobile.jpg) |

Outros pares: tema claro Pearl ([React desktop](screenshots/react-artist-pearl-desktop.jpg),
[Go desktop](screenshots/go-artist-pearl-desktop.jpg), [React celular](screenshots/react-artist-pearl-mobile.jpg),
[Go celular](screenshots/go-artist-pearl-mobile.jpg)); minutos visíveis
([React](screenshots/react-rewind-time-desktop.jpg), [Go](screenshots/go-rewind-time-desktop.jpg));
[gestão React acessada a partir do Go](screenshots/go-react-management-desktop.jpg).

A comparação revelou um estouro horizontal na mídia em 390 px, corrigido com a
largura explícita do conteúdo; o viewport passou de 486 px de largura de rolagem
para 390 px. O cartão da retrospectiva móvel também ganhou imagens menores para
preservar espaço dos títulos. Há diferenças visuais remanescentes: o React usa
um gradiente no cabeçalho de mídia, e o Go não; no rewind o título e os controles
estão dispostos de forma diferente, com botão Apply para a opção de tempo no Go.
Os indicadores e a ordenação verificados no período são iguais. O gráfico Go é
SVG com tabela acessível e não uma captura pixel a pixel do Recharts.

## Limites das verificações realizadas

Ainda não foram reproduzidos manualmente o gesto de arrastar um arquivo para o
editor, todas as combinações de edição/mesclagem de artista e álbum, e o percurso
completo por teclado. A substituição de imagem pela API foi exercitada na
preparação do banco; isso não prova o gesto no navegador. Não se deve tratar estas
capturas como prova de todos os estados de erro nem de paridade visual exata. O
usuário confirmou o aceite e o encerramento em 05/11/2026; estas verificações
continuam registradas como não executadas. Não houve publicação ou push.
