# Issue 036 — admissão e cancelamento de imagens

Concluída em 06/11/2026. [Issue](<../../tasks/issue-036 - 036-—-Admissão-e-cancelamento-de-imagens.md>) ·
[Revisão que motivou a correção](ADVISOR_032_033.md).

## Correções

- Uploads administrativos de capas, avatares e criação de perfil entram em
  admissão antes de o adaptador interpretar o formulário. O serviço também
  aplica a admissão para chamadas diretas; um contexto de serviço derivado do
  adaptador reutiliza a mesma reserva. Há um ativo e até quatro esperas; excesso
  retorna `ErrBusy`/HTTP 503 e a espera pode ser cancelada antes da leitura.
- Esperas pelas travas de perfis e publicação podem ser canceladas. A trava de
  publicação continua protegendo a sequência administrativa de arquivo e
  associação contra a coleta de órfãos.
- Derivação e redownload compartilham trabalho por chave, em grupos separados de
  até cinco trabalhos distintos. Cada cliente pode abandonar sua espera; o
  trabalho tem contexto próprio com prazo de 30 segundos e é cancelado quando o
  último participante sai. Se um codec ainda estiver executando, o grupo conserva
  seu slot até a função terminar, mesmo após cancelamento ou prazo.
- Excesso de bytes no upload ou download administrativo e no limite do formulário
  retorna 413; formatos não suportados chegam à validação por conteúdo e retornam
  400 com orientação. Arquivos inválidos não são associados.

## Validação

- `go test ./...` passou com Go 1.26.3. Após um ajuste posterior para preservar o
  posicionamento do leitor de avatar, os testes específicos de avatar passaram
  novamente, inclusive com `-race`.
- `go test -race ./internal/imagework ./internal/admin ./imagecache ./engine
  ./engine/handlers` passou; `go vet` nos pacotes alterados passou.
- Go 1.25.0 com `CGO_ENABLED=0`: testes de imagework, admin, cache e handlers
  passaram.
- Testes exercitam cinco reservas administrativas, sobrecarga e cancelamento;
  leitor HTTP não acessado antes de uma admissão cancelada; primeiro ou outro
  participante cancelado; cancelamento do último; prazo; manutenção do limite
  enquanto trabalho não interrompível termina; erro de formato e limites HTTP
  do arquivo e do envelope; download remoto acima de 10 MiB; avatar e artwork
  válidos; cache legado, gravação atômica e associação.

## Limites

A admissão administrativa e os grupos de derivação/redownload têm limites
separados; não são uma cota única para toda a memória de rede da aplicação.
Um leitor bloqueado e um codec em execução não são interrompidos à força. O
prazo libera os participantes, mas o trabalho permanece contabilizado até
terminar; uma nova solicitação para a mesma chave nesse intervalo recebe 503.
O limite HTTP de avatar continua contando o envelope multipart. A janela
histórica entre publicação automática do catálogo e coleta foi tratada depois na
[issue 037](<../../tasks/issue-037 - 037-—-Coordenar-publicação-automática-e-coleta-de-imagens.md>).

Não houve mudança de SQLite, limpeza do cache ou build de distribuição. A 034
continua responsável por remover a dependência nativa e validar distribuição.
