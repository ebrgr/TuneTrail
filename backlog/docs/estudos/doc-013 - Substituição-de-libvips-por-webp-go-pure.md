---
id: doc-013
title: Substituição de libvips por webp-go-pure
type: other
created_date: '2026-11-12T23:39:00Z'
updated_date: '2026-11-12T23:39:00Z'
tags:
  - estudo
---
> Importado de [docs-dev/ESTUDO_WEBP_GO_PURE.md](../../../docs-dev/ESTUDO_WEBP_GO_PURE.md).

# Substituição de libvips por webp-go-pure

Atualização de direção em 06/11/2026: o usuário escolheu planejar a combinação
imaging + webp-go-pure. [Plano e issues 031–035](../../../docs-dev/PLANO_TRANSICAO_IMAGENS_GO.md).
As medições exploratórias abaixo continuam sendo do protótipo com draw.

Investigação de 06/11/2026. Relacionada ao [estudo de Go sem CGO](../../../docs-dev/ESTUDO_01_GO_SEM_CGO.md).
Conclusão: **candidato viável para o codec WebP, com trabalho adicional no pipeline
de imagens**. A prova isolada compilou sem CGO; a aplicação ainda não foi migrada.
Não foi feita escolha definitiva nem alteração em `go.mod` do Koito.

## Código examinado e evidência

Foi baixado o repositório oficial para uma pasta temporária, no commit
[`75d274a01bc93d4fc5faeed0e40821fe50757294`](https://github.com/SeriousBug/webp-go-pure/tree/75d274a01bc93d4fc5faeed0e40821fe50757294).
O módulo declara Go 1.22 e não declara outros módulos; a licença é MIT e seus
avisos devem ser preservados. Isso é compatível em princípio com o Go 1.25
declarado pelo Koito, mas a prova abaixo usou Go 1.26.3, não Go 1.25.
Fontes: [go.mod](https://github.com/SeriousBug/webp-go-pure/blob/75d274a01bc93d4fc5faeed0e40821fe50757294/go.mod),
[licença](https://github.com/SeriousBug/webp-go-pure/blob/75d274a01bc93d4fc5faeed0e40821fe50757294/LICENSE).

O pacote `/std` oferece integração com `image.Image` e opções de qualidade,
esforço, modo lossless e EXIF. Não oferece recorte, redimensionamento, conversão
geral de formatos ou gerenciamento ICC. A API raiz permite consultar `Features`
antes de alocar os pixels. Fonte: [implementação std](https://github.com/SeriousBug/webp-go-pure/blob/75d274a01bc93d4fc5faeed0e40821fe50757294/std/webp.go)
e [API do codec](https://github.com/SeriousBug/webp-go-pure/blob/75d274a01bc93d4fc5faeed0e40821fe50757294/docs/codec-api.md).

## Alterações necessárias no Koito

| Arquivo/área | Alteração proposta e motivo |
| --- | --- |
| Novo `internal/imagecodec` ou componente equivalente | Centralizar inspeção de formato, limites, decodificação, orientação, recorte, redução e WebP. Downloads, avatar e capas precisam da mesma política. |
| `imagecache/imagecache.go` | Substituir `bimg.Size` e `bimg.NewImage(...).Process` pelo pipeline Go. Preservar tamanhos, caminhos, MIME, escrita temporária com rename, singleflight e coordenação da publicação/coleta. |
| `internal/admin/artwork.go` | Trocar a validação de dimensões via bimg pelo mesmo componente, evitando regras diferentes entre upload e download. |
| `internal/admin/profiles.go` | Avatar já passa por `SaveImage`; revisar os testes e garantir limite antes de decodificar, inclusive se a chamada não trouxer um reader limitado. |
| `internal/summary/image.go` | Remover/substituir o registro de `golang.org/x/image/webp`, ou fazer dispatch explícito. Dois decodificadores registrados para o mesmo formato tornam a escolha dependente do registro. |
| `go.mod` e `go.sum` | Adicionar versão/commit fixado de webp-go-pure; retirar bimg depois de eliminar todos os usos. `x/image` ainda pode ser usado para escala e fontes, portanto não deve ser removido indiscriminadamente. |
| `Makefile` e `.vscode/launch.json` | Build e execução normais com `CGO_ENABLED=0`; retirar exigência de libvips/pkg-config do caminho normal. Preservar depuração e testes dos scripts Go UI. |
| `Dockerfile` | Compilar sem CGO e retirar libvips-dev, pkg-config e libvips42. Escolher runtime e validar certificados, fusos, permissões e healthcheck antes de reduzir a imagem. Não adotar scratch automaticamente. |
| `.github/workflows/test.yml` e documentação | Testar build/execução sem libvips e plataformas suportadas; atualizar instruções de instalação. O detector `go test -race` ainda exige CGO nas plataformas em que é usado: manter compilador C nesse job específico, sem tornar libvips dependência da aplicação. |

## Comportamentos que exigem decisão ou regressão

**Formatos de entrada.** WebP é o formato do novo codec. JPEG e PNG exigem seus
decodificadores Go; GIF e demais formatos precisam de inventário e política
explícita. Formatos aceitos dependem hoje da instalação da libvips; não presumir
que trocar o encoder preserve automaticamente AVIF, HEIF, TIFF ou SVG. A
verificação de MIME atual não substitui validação de conteúdo e dimensões.

**Orientação, recorte e escala.** A bimg atual autorrotaciona imagens conforme
EXIF. Será necessário tratar rotações e espelhamentos antes do recorte e remover
os metadados depois. Comparar filtro de escala Go com o atual. O `source` usa
lado `min(largura, altura, 1200)` e qualidade 100; derivados pedem 64, 128, 300,
640 e 1000 com qualidade 85, sem `Enlarge`. A prova observou que uma imagem
280×280 solicitada como 300 continua 280×280. Não forçar aumento como efeito
colateral da migração. Testar também imagens pequenas retangulares.

**Qualidade e cores.** Qualidade 100 atual não seleciona `Lossless`; ativar esse
modo seria mudança de comportamento e custo. Os números 85/100 não garantem
mesma imagem entre encoders. Transparência exige cuidado com alpha direto versus
premultiplicado: usar `image.Image` corretamente e não entregar `image.RGBA.Pix`
diretamente à API raiz. EXIF não é correção de cor: decidir tratamento de ICC,
JPEG CMYK e conversão para sRGB, comparando o comportamento observado atual.

**Animações e recursos.** Pela leitura do commit examinado, `std.Decode` chama
`DecodeAnimation` para animações e só depois devolve o primeiro quadro;
`animation.go` mantém cópia do canvas por quadro. A aplicação deve consultar
`Features.HasAnimation` e rejeitar animações antes do decode, ou implementar uma
rota comprovadamente limitada ao primeiro quadro. Essa restrição precisa ser
documentada se diferir da aplicação atual. Fonte: [animation.go](https://github.com/SeriousBug/webp-go-pure/blob/75d274a01bc93d4fc5faeed0e40821fe50757294/animation.go).

Um buffer RGBA de 40 milhões de pixels demanda cerca de 153 MiB, sem contar
outros buffers. A estratégia Go proposta decodifica a imagem antes de reduzir,
enquanto a bimg pode reduzir durante a carga. Limite de bytes e singleflight
por ID/tamanho não limitam o trabalho entre IDs distintos. Prever limite de
concorrência e orçamento de pixels/memória, e medir antes de fixar valores.
Não foi encontrada API pública com `context.Context` no codec examinado; checar
cancelamento entre fases não interrompe uma codificação já em andamento.

**Cache e banco.** É possível preservar os arquivos `.webp`, `image/webp`, UUIDs
e URLs atuais; a troca de codec não exige por si migração SQLite ou limpeza de
cache. Novos derivados podem ter aparência/tamanho diferentes dos antigos.
Testar regeneração a partir do `source.webp` antigo, ausência de origem,
redownload e rollback. Não recodificar nem excluir as imagens do usuário como
parte da prova. O esquema vigente continua v3.

## Prova isolada executada

Ambiente: macOS arm64, Go 1.26.3, dependências existentes do Koito. Dois programas
temporários em `/private/tmp/koito-webp-poc` usaram módulos separados do projeto;
o candidato foi referenciado pela cópia local do commit acima. Nenhum banco foi
aberto e `go.mod`/`go.sum` do Koito permaneceram inalterados.

- `CGO_ENABLED=0 ... go test ./std` passou no repositório do candidato.
- Programa Go sem CGO: JPEG de teste e dois WebP existentes foram decodificados,
  recortados/reduzidos e codificados com qualidade 85 e esforço mais rápido.
  Foram conferidas dimensões; uma amostra sintética preservou o alpha 83.
- O programa rejeitou o arquivo animado da fixture por cabeçalho antes de
  decodificar seus quadros; isso é comportamento da prova, não correção upstream.
- A libvips leu os três WebP gerados pelo candidato; o candidato também leu e
  recodificou três WebP produzidos pela libvips. Cobertura pequena, não garantia
  universal de compatibilidade de arquivos.

Média de três transformações por arquivo, incluindo decode/resize/encode,
excluindo leitura de disco e compilação. A prova Go usou CatmullRom e recorte
central; o pipeline bimg usou as opções atuais. Não são filtros nem codecs
equivalentes e não houve equalização de qualidade visual.

| Fixture / saída | bimg/libvips: tempo e bytes | Go candidato: tempo e bytes |
| --- | --- | --- |
| `test_assets/yuu.jpg`, 280×280 | 6,65 ms; 9.766 bytes | 14,98 ms; 13.252 bytes |
| `test_assets/default_img.webp`, 300×300 | 7,95 ms; 804 bytes | 20,05 ms; 1.616 bytes |
| `internal/webui/static/avatar.webp`, 128×128 | 1,23 ms; 974 bytes | 1,32 ms; 1.752 bytes |

Os números mostram que **não se deve prometer ganho de velocidade ou redução de
arquivos** com base na remoção de CGO. São somente três imagens pequenas e uma
execução exploratória; variam com máquina, aquecimento e parâmetros. Não foram
medidos pico de memória, carga paralela, EXIF/ICC, imagens grandes, PSNR/SSIM ou
aceite visual. O README upstream também recomenda conferir memória no modo
lossless; seus benchmarks não substituem medições no Koito.

## Encaminhamento

Recomendo uma issue local de prova do pipeline completo antes da troca em
produção: contrato de formatos/animação, orientação EXIF 1–8, corpus de cache
antigo, limites de recursos, comparação visual e benchmark em imagens reais
autorizadas ou fictícias representativas. Depois, integração com capas/avatares,
testes concorrentes, build Go sem CGO, Docker sem libvips e verificação de
instalação/upgrade/rollback. A redução de dependências nativas é o benefício
pretendido; custo de CPU, memória e tamanho dos arquivos precisa ser aceito
com medições.

Advisor (Sol, esforço médio): **reprovado com ressalvas para uma troca direta**;
confirmou por leitura as lacunas de transformação, formatos, animação, EXIF,
registro duplo de decoder e recursos. Não executou os testes da prova. A revisão
orientou o plano acima; a biblioteca não foi auditada integralmente.
