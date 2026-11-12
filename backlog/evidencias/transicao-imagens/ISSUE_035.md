# Issue 035 — preparação das imagens da retrospectiva

Concluída em 06/11/2026. [Issue](<../../tasks/iss-035 - 035-—-Refatorar-o-código-de-imagens-da-retrospectiva-com-imaging.md>) ·
[Plano de transição](<../../docs/estudos/doc-014 - Plano-de-transição-—-imaging-webp-go-pure.md>).

## Resultado

O antigo `GenerateImage` era um protótipo incompleto, inteiramente comentado e
sem chamadas ativas. O bloco obsoleto foi substituído por uma nota explícita: a
funcionalidade continua desativada. Não foi criada rota ou opção de compartilhar,
nem alterados os cálculos da retrospectiva ou o HTML existente.

Os auxiliares agora recebem imagem por `io.Reader` e fonte por `[]byte`, sem
caminhos relativos ao diretório atual. O chamador é dono do reader e deve
fechá-lo quando necessário; o auxiliar não abre arquivos. Cada face de fonte
criada é fechada. Não foram incorporadas nem redistribuídas fontes de terceiros:
a fixture usa Go Regular, fornecida pelo pacote `golang.org/x/image/font/gofont`.
`x/image` permanece necessária para fontes e como dependência de imaging.

A imagem é decodificada pela API compartilhada `imagepipeline.WithDecoded`,
com validação de conteúdo, limites, orientação e o mesmo registro WebP usado no
cache. O callback executa enquanto ocupa o mesmo slot global do pipeline e não
deve reter a imagem após retornar. Leitores bloqueados, callbacks e codecs em
andamento não podem ser interrompidos à força. A largura de saída do auxiliar é
limitada a 1200 pixels por lado.

## Semântica de escala e composição

- `imaging.Resize` com `NearestNeighbor` força largura e altura escolhidas;
  imagens retangulares são esticadas, preservando a geometria anterior. Não usa
  `Fit` ou `Fill`. Dimensões negativas retornam nil; zero ou imagem vazia geram
  tela vazia com as dimensões solicitadas.
- A amostragem de imaging usa centros de pixels; a rotina anterior usava
  arredondamento para baixo. Na redução de três pixels para dois, por exemplo,
  a antiga selecionava 0 e 1, enquanto a nova seleciona 0 e 2. É uma diferença
  intencional de conteúdo, não uma promessa de equivalência pixel a pixel.
- A composição mantém `image/draw` com `draw.Over`: o ponto é uma coordenada da
  **fonte**, não a posição de destino. Isso preserva a operação antiga em
  pontos positivos/negativos, clipping e canvas de origem não zero. A escala
  lida corretamente com bounds de origem não zero, corrigindo a rotina antiga.
- Não há gerenciamento de perfis ICC ou garantia colorimétrica geral nova; o
  comportamento de formatos e cor é o mesmo da [issue 032](ISSUE_032.md).

## Testes e inspeção

- Suíte Go completa passou com CGO desativado; testes de summary e pipeline
  passaram no Go 1.25.0 sem CGO e com `-race` no ambiente local. `go vet` passou.
- Testes verificam dimensões esticadas, amostragem literal, bounds não iniciados
  em zero, alpha, pontos positivos/negativos, clipping, formatos, WebP, entradas
  inválidas, contexto, mensagens de callback e compartilhamento da capacidade.
  O teste de fonte executa em diretório temporário vazio.
- A regressão do corpus 031 comparou **61 saídas** do pipeline de capas com os
  resultados anteriores e passou; não houve divergência nesses casos no ambiente
  Go 1.26.3. Apenas o decoder WebP de `webp-go-pure` aparece em `go list -deps`.
- [Composição fictícia](issue-035/composition-035.png) e
  [comparação de amostragem](issue-035/sampling-035.png) foram geradas por testes
  e inspecionadas. São artefatos de teste, não capturas da interface ou proposta
  de layout final.

Para reproduzir as imagens em uma pasta já existente:

```sh
KOITO_SUMMARY035_OUTPUT=/caminho/para/saida CGO_ENABLED=0 \
  go test -count=1 -run '^Test(Composition|SamplingComparison)035$' ./internal/summary
```

## Próxima funcionalidade

Uma funcionalidade futura de imagens sociais precisa de especificação própria:
layout e formatos de exportação, seleção de dados e privacidade, fontes e suas
licenças, fornecimento seguro dos recursos, limites e acessibilidade, além de
jornadas de interface. Esta issue não ativa esse gerador.
