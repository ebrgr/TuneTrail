# Issue 034 — distribuição sem CGO e libvips

Validação em 06/11/2026. [Issue](<../../tasks/issue-034 - 034-—-Distribuir-e-validar-a-aplicação-sem-CGO-libvips.md>) ·
[Plano](<../../docs/estudos/doc-014 - Plano-de-transição-—-imaging-webp-go-pure.md>) ·
[Contrato e medições](ISSUE_031.md) · [Pipeline](ISSUE_032.md) ·
[Integração e imagens](ISSUE_033.md) · [Correções após revisão](ISSUE_036.md).

## Alterações

Nenhum pacote da aplicação importa bimg; a dependência foi retirada do módulo
raiz com `go mod tidy`. `golang.org/x/image` continua necessário para fontes e
pela imaging. O módulo independente em `scripts/experiments/images031` mantém
bimg/libvips apenas para reproduzir o baseline histórico, fora do build normal.

Build, execução, vet, testes normais e prévia usam `CGO_ENABLED=0`; o alvo
opcional `make api.race` usa CGO e requer compilador C. Docker não instala
libvips ou pkg-config: mantém Debian Bookworm, CA, tzdata, curl, healthcheck,
usuário e `ENTRYPOINT` existentes. VS Code/Delve, CI, documentação de
instalação e deploy foram atualizados. A CI compila Linux amd64/arm64 e testa o
contêiner também em PR e chamadas do workflow; o teste race mantém CGO.

## Validação

- Binários Linux amd64 e arm64 construídos com Go 1.25.0 e CGO desativado;
  `file` identificou ambos como ELF estático. A arquitetura executada deve ser
  distinguida da simplesmente compilada, conforme os resultados abaixo.
- `CGO_ENABLED=0 go test ./...` passou no macOS arm64 com Go 1.26.3;
  `make check` passou em cópia isolada sem `.env`. Após o ajuste do alvo de UI,
  `make ui.check` foi repetido na mesma cópia. O job de race da CI passou com
  Go 1.25.0 e CGO habilitado em SQLite/engine.
- Binário macOS Go 1.25 sem CGO iniciou em diretório vazio, serviu páginas e
  assets incorporados e criou SQLite v3 íntegro. As jornadas multiusuário, com
  ingestão, importação/exportação, isolamento e reinício, passaram em bancos
  novos e migrados da v2. O encerramento com relay pendente preservou a escuta
  e a integridade; entrega de relay em andamento não é garantida.
- Delve 1.27.2 iniciou com binário de símbolos e CGO=0; breakpoint em
  `main.main` foi atingido. O teste automático da configuração F5 também passou.
- Compose validou a configuração. Os contêineres arm64 e amd64 foram criados
  separadamente com volumes novos:
  `healthy` pelo Docker, endpoint HTTP, ausência de pacote libvips, CA e fuso
  instalados, conversão de DST São Paulo, HTTPS com curl e programa Go sem CGO,
  parada por SIGTERM com código zero e SQLite v3 íntegro. Após reinício, um
  perfil fictício criado antes da parada foi encontrado com o mesmo ID. O
  contêiner e o volume de teste foram removidos pelo script.

Ambos os testes de contêiner passaram, incluindo o probe Go de HTTPS. O host
Docker era arm64; a execução amd64 ocorreu sob emulação. A CI executa a imagem
amd64 em runner Linux e compila ambas as arquiteturas; não executa arm64 nela.
`go mod tidy -diff` com Go 1.25.0 não apontou alterações.

## Qualidade, cache e rollback

As imagens mantêm os resultados das 031–033: parâmetros e limites, custos e
comparações estão nos relatórios ligados acima. A 033 verificou leitura de WebP
legado pelo pipeline novo e leitura de source e derivado novos pela libvips
anterior, em dados fictícios. Preservar backup do diretório SQLite/cache antes de
atualizar. Rollback entre versões só é seguro quando a versão anterior aceita o
esquema e os dados: se houver migração incompatível, restaurar o backup; não
executar downgrade nem limpar ou recodificar o cache automaticamente.

## Advisor e limites

Advisor (Sol, esforço médio) revisou Docker, build, CI, debug, certificados,
fuso, permissões e sinais. Parecer final **aprovado** após corrigir o teste de
persistência, padronizar CGO=0 nas tarefas UI e executar o job de contêiner em
PR/workflow_call. O parecer é revisão do desenho e arquivos, não uma execução
independente de todos os testes. O serviço ainda executa como root no contêiner,
como antes; alterar permissões e ownership de volumes exigiria trabalho próprio.

A issue histórica [037](<../../tasks/issue-037 - 037-—-Coordenar-publicação-automática-e-coleta-de-imagens.md>)
estava aberta ao concluir esta distribuição e foi resolvida depois. A refatoração da imagem de
retrospectiva permanece na 035.

## Reprodução

```sh
CGO_ENABLED=0 go test ./...
CGO_ENABLED=1 go test -race ./internal/db/sqlite ./engine -run 'Test(Profile|Consolidated|AdminAccount)' -timeout 120s
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /private/tmp/koito-amd64 ./cmd/api
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o /private/tmp/koito-arm64 ./cmd/api
CGO_ENABLED=0 go build -o /private/tmp/koito-local ./cmd/api
python3 scripts/validate_standalone_030.py --binary /private/tmp/koito-local
python3 scripts/validate_multiuser.py --binary /private/tmp/koito-local
docker build -t koito-sum-test:local .
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o /private/tmp/koito-tls ./scripts/experiments/tls034
python3 scripts/validate_container_034.py --image koito-sum-test:local --tls-probe /private/tmp/koito-tls
```

O `GOARCH` do probe precisa corresponder ao contêiner. Os scripts criam dados
fictícios em diretórios ou volumes novos e removem os contêineres/volumes que
criam. O teste HTTPS externo requer rede e uma CA válida.
