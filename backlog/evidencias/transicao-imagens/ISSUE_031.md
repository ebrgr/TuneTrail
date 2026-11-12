# Issue 031 — contrato e prova da combinação imaging + webp-go-pure

Execução em 06/11/2026. [Plano](<../../docs/estudos/doc-014 - Plano-de-transição-—-imaging-webp-go-pure.md>) ·
[contrato e critérios](CONTRATO_031.md) · [módulo reproduzível](../../../scripts/experiments/images031/README.md).
Esta entrega define a transição e seus parâmetros; o pipeline de produção ainda
usa bimg/libvips. Nenhum banco, `.env` ou cache operacional foi lido/alterado.

## Entrega e versões

- **184 casos**, incluindo dois probes adicionais de eixo 9.999 fora da medição
  de tempo; 21 recusas esperadas e nenhum resultado inesperado no classificador.
- **61 pares** de transformações válidas, dimensões iguais; leitura cruzada de
  origens WebP antigas/novas nos dois pipelines e derivados 64/128/300/640/1000.
- **138 saídas** com qualidade medida, incluindo derivados Q85 e esforços2/4.
  Dimensões, alpha, metadados, limites e concorrência têm evidências separadas.
- [Manifesto de 38 fixtures](corpus-manifest.json),
  [resultados brutos](results-031.json) e [classificação dos gates](summary-031.json).
  O corpus grande é regenerado em pasta temporária para evitar arquivos artificiais
  de 10 MiB/40 MP no repositório; gerador e hashes foram preservados.

Ambiente: macOS 15.7.3 arm64, Go1.26.3, GOMAXPROCS=2 e VIPS_CONCURRENCY=2.
bimg v1.1.9, libvips8.18.7, imaging v1.6.2 e webp-go-pure
v1.2.1-0.20260809063125-75d274a01bc9. Os três módulos são MIT; avisos de copyright
constam nas licenças upstream e devem acompanhar redistribuição quando aplicável.
`x/image v0.33.0` permanece somente no módulo experimental para acompanhar a
versão atual do Koito; o decoder WebP antigo não é importado pelo worker Go.
A análise de segurança e a validação no Go1.25 declarado são requisitos da 032,
não demonstrados pelo uso local de Go1.26.3.

## Corpus e contrato observado

São imagens procedurais, não fotografias: texturas de estresse, um campo de tons
naturais, gradientes, marcadores assimétricos e transparência. EXIF1–8 foi
injetado em JPEG fictício; ICC RGB foi gerado como matriz/TRC de teste; CMYK foi
criado localmente pelo ImageMagick7.1.2-34. Nenhum perfil ICC de terceiros ou
imagem pessoal foi copiado. Esses casos não são calibração colorimétrica geral.

O corpus inclui JPEG/PNG, WebP legado produzido pela bimg, GIF estático/animado,
WebP animado, BMP, TIFF, SVG, HEIC/AVIF, arquivo inválido e JPEG truncado.
Limites de 10 MiB−1/exato/+1, eixos 9.999/10.000/10.001, 40 MP exatos e
8000×4999 / 8000×5001 foram exercitados. Os limites de pixels ±1 não se isolam
em um par de lados inteiros ambos <=10.000; não foram rotulados como tal.
`zero.png` é cabeçalho inválido. Probes de admissão não equivalem a decode válido.

| Observação | Resultado e consequência |
| --- | --- |
| small.png 80×48, pedido 300 | Ambos preservaram 80×48. Não trocar por Fill(300,300) indiscriminadamente. |
| thin.png 100×500, pedido 300 | Ambos produziram 300×300. O ramo Enlarge=false da bimg só evita ampliação quando ambos os lados são menores; preservar essa exceção ou registrar futura mudança. |
| source e derivados | Source: lado mínimo limitado a1200, Q100 lossy. Derivados Q85, mesmos tamanhos/nomes/MIME. Lossless não foi ativado. |
| EXIF1–8 | Teste Go comparou cada pixel com regras literais de rotação/espelhamento; as saídas source também tiveram comparação com oráculo independente. |
| Alpha e bounds | Testes Go passaram com origem não zero e alpha após transformação. O alpha de uma saída WebP foi decodificado também pelo ImageMagick e comparado ao PNG pré-encode: erro máximo zero. |
| Metadados | Nenhuma saída WebP examinada propagou chunks EXIF/ICCP/XMP; as entradas fictícias tinham EXIF, comentário ou perfil conforme o caso. |

### Formatos e limites — política proposta, ainda não aplicada

A libvips local processou JPEG, PNG, WebP estático, GIF estático/animado, WebP
animado, TIFF, SVG, HEIC e AVIF. BMP foi recusado pela bimg neste ambiente.
Capacidades declaradas de PDF/MAGICK aparecem no JSON, mas não comprovam upload
aceito: o sniff de conteúdo dos endpoints e limites dos chamadores também contam.

O contrato proposto da 032 limita o produto a JPEG, PNG e WebP estático por
conteúdo. A recusa dos outros formatos é uma **restrição de suporte planejada**,
não uma equivalência já comprovada; a implementação precisa produzir mensagem
clara e documentar a mudança. WebP animado é recusado antes de decodificar quadros.
A imaging registra BMP/TIFF/GIF, portanto a allowlist deve anteceder o decode.

Downloads/capas já têm teto10MiB e capas/downloads validam lados/pixels. Avatar
tem limite HTTP5MiB, incluindo multipart. SaveImage/compressImage atuais leem
sem limite interno próprio; o worker aplica teto adicional experimental e isso
não deve ser apresentado como proteção universal da aplicação atual.
O novo componente deverá limitar readers na entrada, antes de alocar pixels.

## Medições de transformação e recursos

Os tempos abaixo incluem decode/config interno, orientação, corte/escala e
encode. **Excluem leitura do arquivo e inspeção/admissão externa**, além de startup
do processo. O RSS inclui todas essas fases e memória Go+nativa. Cada amostra
usa processo independente; as cargas1/2/4 acontecem dentro do mesmo processo,
compartilhando a entrada já carregada. Buffers HTTP independentes não foram medidos.

São três repetições simples; o campo p95 é o maior valor dessas três, não uma
estimativa estatística de produção. Na carga há duas operações por worker.

| Origem | bimg: máximo observado / bytes | Candidato esforço−1: máximo / bytes |
| --- | --- | --- |
| Campo natural JPEG → source480 | 20,48ms /84.240 | 27,47ms /100.068 |
| Alpha PNG → source240 | 9,99ms /33.386 | 15,02ms /40.960 |
| Textura2400×1600 → source1200 | 140,17ms /1.077.268 | 197,82ms /1.299.892 |

Nos61 pares com esforço−1, o pior fator de tamanho foi **1,455×** e de tempo
**2,289×**, abaixo dos alertas iniciais1,5×/3×. Não são garantias para outros
conteúdos, máquinas ou caminhos HTTP. Esforços2/4 também foram medidos com
oráculos: reduzir bytes não trouxe melhora uniforme de qualidade e aumentou CPU.

| Carga Go, entrada2400×1600 | Operações / vazão | Pico RSS |
| --- | --- | --- |
| 1worker | 2 /4,88op/s | 54,9MiB |
| 2workers | 4 /8,80op/s | 107,4MiB |
| 4workers | 8 /8,76op/s | 204,1MiB |

Uma transformação Go de **40MP** atingiu306,5MiB, contra168,2MiB da bimg.
É uma operação sintética, não endurance. Gates768MiB para40MP e512MiB para
carga2 passaram neste ambiente. A recomendação conservadora para032 é **um
processamento global, quatro esperas canceláveis e nenhuma imagem decodificada
na espera**. Aumentar concorrência só depois de medir hardware/carga de destino.

## Qualidade e diferenças explicitadas

O oráculo PNG usa geometria independente no ImageMagick, sem codec com perdas:
original → source PNG → derivado PNG. RGB é comparado composto sobre branco e
preto, com alpha separado. Isso mede também as duas compressões quando aplicável.
Os filtros nativos/Go não são idênticos; PSNR não substitui julgamento visual.

O gate absoluto inicial de30/28dB gerou **122 alertas** no corpus de estresse,
incluindo o próprio baseline. Esses resultados foram preservados; não se declara
aprovação integral do gate inicial. Campo natural source: baseline46,33dB e
candidato45,06dB no JPEG, diferença de−1,274dB. O PNG teve diferença de−1,172dB.
Nos derivados naturais, ambos ficaram entre38,87 e39,53dB.

O contrato final classifica os conteúdos: natural source>=40dB, derivado>=35dB,
com tolerância relativa1,5dB na origem e0,5dB nos derivados. Estresse usa referência
relativa de0,5dB com inspeção e medidas absolutas registradas. Esses parâmetros
foram **calibrados após a prova**, não são gates originais aprovados. Não houve
falha desses gates calibrados nas61 comparações. A fidelidade de fotografias,
perfis ICC reais e JPEG CMYK de outros produtores ainda precisa ser ampliada.

Inspeção visual das amostras mostrou texturas e orientação coerentes; CMYK teve
diferença de saturação entre pipelines. Não atribuir correção universal a um
deles: a referência CMYK não é calibrada. A 032 deve explicitar a política de
cores e manter esse caso como regressão documentada.

| Amostra | bimg | imaging + codec |
| --- | --- | --- |
| Campo natural, origem | [baseline](vips-natural.png) | [candidato](pure-natural.png) |
| Campo natural, derivado300 | [baseline](vips-derived-natural.png) | [candidato](pure-derived-natural.png) |
| Transparência, derivado300 | [baseline](vips-derived-alpha.png) | [candidato](pure-derived-alpha.png) |
| EXIF6, origem | [baseline](vips-exif-6.png) | [candidato](pure-exif-6.png) |
| CMYK, origem | [baseline](vips-cmyk.png) | [candidato](pure-cmyk.png) |
| ICC sintético, origem | [baseline](vips-icc.png) | [candidato](pure-icc.png) |

Algumas prévias foram reduzidas a320px somente para inspeção; as métricas usam
as saídas originais. Essas imagens são comparação do processamento, não capturas
do aplicativo. As jornadas de UI/cache integram o escopo da033.

## Encaminhamento para032

Parâmetros iniciais: imaging/Lanczos, webp EffortFastest(−1), sourceQ100 e
derivadosQ85, lossy. Preservar contratos de cache/HTTP/SQLitev3 e implementar os
limites globais e de entrada antes de integrar. As restrições de formato,
diferenças de cor e limites de cancelamento devem ser explícitos.

Passaram os testes do módulo (bounds, alpha, EXIF1–8), inclusive com `-race`,
build do candidato sem CGO, vet do experimento e classificador de resultados. Não houve migração do
pipeline, alteração de go.mod/go.sum raiz, teste de carga HTTP, garantia de
cancelamento durante encode nem prova geral de segurança/cores. Repetição no
Go declarado e plataformas de distribuição fica nas032–034.

## Parecer final do Advisor

**APROVADO para encerrar a 031 como estudo e definição de contrato**, em
06/11/2026. Revisão estática dos scripts, contrato, relatório e resultados; o
Advisor não repetiu o benchmark. As ressalvas sobre derivados/esforços, alpha
independente, escopo da latência e classificação de gates foram incorporadas.
Não houve bloqueador remanescente neste escopo.

Permanecem para a 032: validar Go1.25 e plataformas; ampliar fotografias e
ICC/CMYK; implementar limites, capacidade global, espera e cancelamento; medir
buffers HTTP e hardware de destino; documentar restrições de formato e erros;
revalidar critérios calibrados. O parecer não aprova antecipadamente a 032.
