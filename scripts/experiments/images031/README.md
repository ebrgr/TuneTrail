# Experimento isolado da issue 031

Este módulo não é importado pela aplicação. `go test ./...` na raiz ignora seu
módulo aninhado. As dependências, licença MIT das bibliotecas e checksums estão
fixados no go.mod/go.sum deste diretório; os arquivos da aplicação não mudam.

Requisitos da prova: Go compatível com go.mod, Python3, ImageMagick, libvips/bimg
para o baseline e compilador C. O candidato é compilado com CGO_ENABLED=0.
O uso dessas ferramentas na prova não é dependência proposta para o produto.
`x/image v0.33.0` foi mantido para acompanhar o projeto; BMP/TIFF são registrados
pela imaging mas a política experimental os recusa antes do decode. A escolha
final da versão deve passar por análise de segurança na 032.

```sh
# Resolver/verificar módulos apenas no experimento:
(cd scripts/experiments/images031 && go mod download)
# --work-dir precisa ser um diretório NOVO, fora dos dados operacionais:
python3 scripts/experiments/images031/run.py \
  --work-dir /private/tmp/koito031-nova-execucao \
  --report-dir backlog/evidencias/transicao-imagens
python3 scripts/experiments/images031/analyze.py backlog/evidencias/transicao-imagens
(cd scripts/experiments/images031 && CGO_ENABLED=0 go test ./cmd/pure)
```

O gerador cria corpus determinístico com hashes, ICC RGB sintético e CMYK gerado
localmente, sem imagem de pessoa ou biblioteca do usuário. Fixtures grandes e
padded bytes ficam apenas no diretório temporário. Fontes/versões das ferramentas
ficam no JSON. O script nunca abre bancos ou lê `.env`; servidores não são usados.

`pure` mede imaging/Lanczos + codec esforço -1/2/4; `vips` reproduz opções de
produção, com inspeção e teto de bytes/pixels adicionais do contrato experimental.
Não confundir esses controles do worker com uma proteção já existente em todos
os caminhos da API. `metrics` usa um PNG geométrico independente do ImageMagick
para PSNR sobre branco/preto e alpha; a bimg nunca é importada nesse executável.

Cada amostra é um processo independente; os workers de carga 1/2/4 compartilham
o mesmo processo e entrada. Os tempos internos excluem execução do processo e
I/O em disco e inspeção/admissão externa; RSS inclui Go e alocações nativas e é normalizado em bytes. São
poucas repetições de laboratório, não p95 de produção ou teste de endurance.

Os binários da medição são temporários; o gerador e o módulo preservam o método.
Metadados da máquina não incluem `.env`, tokens ou dados pessoais. Não usar
runtime com `-race` para comparar latência; os testes race devem ser separados.

`run.py` chama `quality.py` para avaliar origens, derivados Q85 e esforços2/4
com referências PNG sem perdas. `analyze.py` classifica gates estruturais,
qualidade e recursos; sucesso do comando não aprova todos os gates de qualidade.
Alertas iniciais e critérios calibrados depois da prova são preservados separados.
