# Issue 033 — cache, capas e avatares com pipeline Go

Concluída em 06/11/2026. [Issue](<../../tasks/iss-033 - 033-—-Integrar-pipeline-Go-ao-cache-capas-e-avatares.md>) ·
[Contrato](CONTRATO_031.md) · [Componente](ISSUE_032.md).

## Alterações

Downloads, `SaveImage`, geração de derivados e uploads de capa/avatar usam
`internal/imagepipeline`. A validação antiga com bimg foi removida do cache e do
serviço de artwork; nenhum caminho Go operacional importa bimg. A dependência
continua declarada até a limpeza de build/distribuição da 034.

O formato de URLs e nomes, MIME `image/webp`, UUIDs, tamanhos, imagens existentes,
`singleflight`, gravação temporária com rename e trava compartilhada com o
coletor de órfãos foram preservados. A leitura da source para um derivado agora
ocorre dentro do trabalho compartilhado, que verifica novamente se outro pedido
já publicou o arquivo. Limite direto de 10 MiB é aplicado pelo pipeline; os
limites HTTP anteriores de avatar (5 MiB incluindo multipart) e artwork
permanecem. Não houve alteração de esquema SQLite nem recodificação do acervo.

Aceitam-se JPEG, PNG e WebP estático. GIF/APNG/WebP animado e demais formatos
são recusados, inclusive em downloads. Isto restringe formatos aceitos pela
libvips anterior e está descrito nos formulários, documentação e changelog.
Erros de formato/animação retornam mensagem específica e HTTP 400; excesso de
bytes no pipeline retorna 413; falta de capacidade retorna 503. Falhas de
associação não publicam referência inválida, e arquivos temporários incompletos
não substituem resultados anteriores.

O contexto é respeitado na espera do pipeline, entre etapas e imediatamente
antes da publicação. Um reader bloqueado ou codec já executando não é
interrompido; chamadores HTTP usam os limites/timeouts próprios. Pedidos do mesmo
ID/tamanho usam `singleflight.Do`: quem aguarda o mesmo trabalho não cancela a
espera de maneira independente e recebe o resultado do primeiro chamador. A propagação do contexto HTTP para a geração de derivados é nova em relação
ao antigo GetImage: cancelar o primeiro pedido pode fazer outros participantes
receberem erro mesmo com contexto válido. Isso deve ser considerado em eventual
refinamento.

## Verificação

- `go test ./...` passou no Go 1.26.3; `go vet` dos pacotes alterados passou.
  `go test -race` passou para cache, pipeline, administração e engine.
- Go 1.25.0, `CGO_ENABLED=0`: testes de cache, pipeline e handlers passaram.
  A retirada formal da dependência, CI, Docker e builds distribuídos cabem à 034.
- Testes usam arquivo WebP procedural produzido pela libvips; 20 solicitações do
  mesmo derivado e múltiplos IDs diferentes, limite de reader, entradas inválidas,
  cancelamento e falha de escrita preservam o resultado anterior. A suíte existente
  cobre download cancelado/fallback, autenticação, associação, falha de banco e
  coordenação com coleta de órfãos. O fluxo HTTP de avatar confirma MIME WebP e
  recusa formato não suportado com mensagem. Foram usados bancos de teste.
- O worker da 031 com bimg/libvips leu e transformou tanto uma nova source como
  um novo derivado gerados pelo pipeline (verificação local de rollback, sem
  mudar o cache operacional). A leitura inversa de source legado tem teste fixo.
- Dois servidores com cópias fictícias independentes do SQLite v3, mesma source
  WebP legada e derivados gerados respectivamente por libvips e Go foram usados
  para oito capturas em desktop 1440×1000 e celular 390×844. Foi verificado o
  carregamento das imagens e `integrity_check`/`foreign_key_check` dos bancos.
  A inspeção mostra avatar e capas carregados em ambos; as capturas comparam os
  derivados no mesmo aplicativo Go, não versões distintas do aplicativo.

| Tela | Derivado libvips | Derivado Go |
| --- | --- | --- |
| Perfis, desktop | [imagem](issue-033/033-legacy-profiles-desktop.png) | [imagem](issue-033/033-go-profiles-desktop.png) |
| Home, desktop | [imagem](issue-033/033-legacy-home-desktop.png) | [imagem](issue-033/033-go-home-desktop.png) |
| Perfis, celular | [imagem](issue-033/033-legacy-profiles-mobile.png) | [imagem](issue-033/033-go-profiles-mobile.png) |
| Home, celular | [imagem](issue-033/033-legacy-home-mobile.png) | [imagem](issue-033/033-go-home-mobile.png) |

[Detalhes das capturas](issue-033/screenshots-033.json) ·
[recorte libvips](issue-033/legacy-128.png) · [recorte Go](issue-033/go-128.png).
A comparação dos derivados 128×128 da fixture produziu PSNR de 40,52 dB e alpha
idêntico. Trata-se de uma imagem procedural; não prova equivalência de cor de
fotografias ou perfis ICC/CMYK. Os limites de recursos da 032 continuam válidos
apenas para a máquina e carga medidas. O estudo de distribuição segue na 034.

## Reprodução das capturas

Com a fixture/worker libvips do experimento 031 e Playwright disponíveis:

```sh
go build -o /private/tmp/koito033-server ./cmd/api
PLAYWRIGHT_MODULE=/caminho/para/playwright python3 scripts/validate_images_033.py \
  --binary /private/tmp/koito033-server --work-dir /private/tmp/koito033-novo \
  --legacy-source /caminho/para/source-legada.webp \
  --legacy-worker /caminho/para/worker-vips --output /private/tmp/koito033-capturas
```

A pasta `--work-dir` deve ser nova; o script encerra os dois servidores ao fim.

## Revisão posterior do Advisor

[Parecer de 06/11/2026](ADVISOR_032_033.md): **reprovado com ressalvas**, sem
bloqueador comprovado para iniciar a 034. O limite de um trabalho e quatro
esperas é interno ao pipeline: buffers e esperas anteriores nas travas de
administração não são abrangidos. Excesso de bytes detectado nas guardas do
serviço de artwork pode retornar 400 antes do mapeamento 413 do pipeline;
formatos recusados pelo sniff anterior podem ter mensagem genérica. A proteção
de publicação/associação é administrativa; o parecer registra uma janela
histórica nos produtores automáticos do catálogo.
