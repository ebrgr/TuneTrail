---
id: doc-015
title: Limpeza de artefatos antigos de libvips
type: other
created_date: '2026-11-12T23:39:00Z'
updated_date: '2026-11-12T23:39:00Z'
tags:
  - estudo
---
> Importado de `docs-dev/LIMPEZA_LIBVIPS.md`. Original removido em 12/11/2026; consulte o histórico do Git.

# Limpeza de artefatos antigos de libvips

Em 06/11/2026, após a migração das issues 031–037, a inspeção do projeto
identificou que o executável local ignorado `./koito` ainda era uma compilação
antiga vinculada a `libvips.42.dylib`. Ele foi substituído por uma compilação do
código atual com Go 1.25.0 e `CGO_ENABLED=0`. Uma cópia temporária do executável
anterior foi preservada fora do repositório antes da troca.

Foram removidos quatro executáveis de prévias antigas, também vinculados à
libvips e sem referências nos scripts ou código atuais:

- `.local/koito-webui-final`
- `.local/koito-issue-007`
- `.local/koito-webui`
- `.local/koito-media-006`

O novo `./koito` foi inspecionado com `otool -L` e `go version -m`: não depende
de libvips e informa CGO desativado. A inicialização, rotas e SQLite v3 foram
verificados com o teste de binário isolado, e passaram os testes de pipeline e
compatibilidade com WebP legado. Nenhum banco ou cache operacional foi alterado.

Permanecem, por terem uso explícito:

- `test_assets/images033/legacy-small.webp`, utilizado por teste automatizado de
  leitura de cache anterior; o arquivo é WebP, não uma biblioteca executável.
- `scripts/experiments/images031`, módulo isolado que ainda usa bimg/libvips
  quando alguém reproduz os benchmarks e o baseline histórico. Não integra a
  aplicação, o módulo Go principal, a imagem Docker ou os testes padrão.
- Relatórios e imagens comparativas em `docs-dev`, preservados como evidências
  da migração e compatibilidade. A checagem de ausência do pacote libvips no
  teste de contêiner e na CI permanece intencional.

A instalação de libvips do sistema não foi removida: ela pode ser utilizada por
outros programas e pelo experimento histórico.
