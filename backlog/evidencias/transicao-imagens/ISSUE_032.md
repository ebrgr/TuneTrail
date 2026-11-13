# Issue 032 — componente Go para imagens

Concluída em 06/11/2026. [Issue](<../../tasks/issue-032 - 032-—-Implementar-pipeline-Go-com-imaging-e-webp-go-pure.md>) ·
[Contrato da 031](CONTRATO_031.md) · [Resultados da 031](ISSUE_031.md).

## Implementação

`internal/imagepipeline` recebe um `io.Reader`, limita a leitura a 10 MiB,
verifica a assinatura antes de escolher o decoder, examina o cabeçalho e limita
cada lado a 10.000 e a área a 40 milhões de pixels. Depois de decodificar,
confere novamente dimensões e respeita orientação JPEG EXIF 1–8. Aceita JPEG,
PNG e WebP estático; rejeita GIF, BMP, TIFF e outros formatos, WebP animado e
APNG. Essas regras ainda não estão conectadas aos endpoints.

A saída é WebP sem metadados, com Lanczos, EffortFastest e qualidade 100 para
source e 85 para derivados. O corte preserva o retângulo quando ambos os lados
são menores que o tamanho pedido; quando apenas um é menor, pode ampliar, como
observado na bimg. JPEG/PNG/WebP sem perfil ICC são tratados pelos decoders Go;
não se oferece conversão ICC nem garantia de equivalência de cor CMYK. O size
zero indica source e os demais tamanhos aceitos vão de 1 a 1200 pixels.

Um semáforo global admite um trabalho ativo e quatro esperas, antes de ler o
input; chamadas adicionais retornam `ErrBusy`. A espera é cancelável e o contexto
é verificado entre etapas. Leitor bloqueado e decode/resize/encode em andamento
não são interrompidos pelo componente. Chamadores devem estabelecer timeouts no
transporte e não podem esperar interrupção imediata do codec.

O decoder WebP antigo no summary foi substituído pelo da combinação escolhida;
`go list -deps ./...` encontrou somente o novo registro WebP. A bimg continua no
cache até a 033. A 033 precisa integrar limites por chamada (como os 5 MiB do
avatar), mapear erros HTTP e documentar a redução de formatos antes de ativá-la.
A issue 034 ainda precisa remover libvips e verificar builds de distribuição.

## Dependências e segurança

`imaging v1.6.2` e `webp-go-pure v1.2.1-0.20260809063125-75d274a01bc9`
foram fixados. A análise `govulncheck` detectou vulnerabilidades de BMP/TIFF em
`x/image v0.33.0`, decoders que `imaging` registra. A primeira versão da
inspeção genérica poderia alcançá-los. A implementação final seleciona JPEG e
PNG por assinatura e usa seus `DecodeConfig` específicos; TIFF/BMP são recusados
antes de qualquer decoder. Atualizamos `x/image` para v0.43.0, corrigida e com
Go 1.25 como mínimo; a resolução também atualizou `x/text` para v0.38.0 e
`x/sync` para v0.21.0.

`govulncheck` não identificou vulnerabilidades alcançadas por
`./internal/imagepipeline` com Go 1.26.3 ou Go 1.25.0, na consulta de 06/11/2026.
Ele reportou, respectivamente, 2 e 4 vulnerabilidades em pacotes importados e
26 e 57 em módulos requeridos sem caminho alcançado nessa análise; esses números
não significam que todos os pacotes da aplicação foram auditados.

## Testes e resultados

- `CGO_ENABLED=0 go test ./internal/imagepipeline`, incluindo corpus, no Go
  **1.25.0** e no Go 1.26.3; `go test -race` e `go vet` no componente passaram.
- `go test ./...` passou no Go 1.26.3, com CGO/libvips e sockets locais para os
  testes já existentes. O build completo sem CGO ainda depende da 034.
- Testes cobrem orientação 1–8, alpha, geometria, limites de bytes e dimensões,
  truncamento, GIF/BMP/TIFF/APNG/WebP animado, sobrecarga e cancelamento de fila.
  As comparações de bounds e alpha independentes do estudo 031 continuam válidas
  para as mesmas saídas; nenhuma prova geral de segurança ou cor foi afirmada.
- Com Go 1.26.3, 61 saídas do componente foram idênticas aos hashes da 031.
  No Go 1.25.0, 50 foram idênticas e 11 JPEGs diferentes, com pelo menos
  **49,16 dB** entre resultados decodificados. As 61 saídas de Go 1.25 foram
  reavaliadas contra os mesmos PNGs independentes: **zero falhas dos critérios
  calibrados**. [Resultados por caso](quality-032-go125.json). Os alertas
  absolutos originais da 031 permanecem; os limites foram calibrados a posteriori.
- macOS 15.7.3 arm64, Go 1.26.3, GOMAXPROCS=2: teste isolado de 40 MP atingiu
  pico RSS **323.436.544 bytes** (~308,5 MiB); dois chamadores de 2400×1600,
  serializados pelo componente, atingiram **49.807.360 bytes** (~47,5 MiB).
  Ambos ficaram abaixo dos limites de 768 e 512 MiB da prova. Essas medições não
  incluem múltiplos buffers HTTP nem substituem teste de carga em produção.

O corpus é procedural e não demonstra fidelidade de fotografias ou de perfis
ICC/CMYK reais. As comparações de interface e cache ficam na 033.

## Reprodução

Gerar novamente o corpus, binários de referência e saídas da 031 com as
[instruções do experimento](../../../scripts/experiments/images031/README.md)
em diretórios temporários. Com o caminho resultante em `KOITO_IMAGE031_WORK`:

```sh
KOITO_IMAGE031_WORK=/private/tmp/koito031-nova-execucao CGO_ENABLED=0 go test -v ./internal/imagepipeline
CGO_ENABLED=1 go test -race ./internal/imagepipeline
go vet ./internal/imagepipeline
```

Para reavaliar a qualidade, definir também `KOITO_IMAGE032_OUTPUT` como uma
pasta vazia existente ao executar `TestCorpus031`; em seguida executar
`verify032.py --work-dir <corpus> --outputs <saidas> --report <json>`. A leitura
dos PNGs de referência requer os arquivos produzidos pela execução da 031.

## Revisão da integração

O [parecer Advisor das 032/033](ADVISOR_032_033.md) registra ressalvas de
admissão externa ao componente e contexto do trabalho compartilhado, sem
bloqueador comprovado para iniciar a 034. A revisão não constitui aprovação dos
builds de distribuição.
