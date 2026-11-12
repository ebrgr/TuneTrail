---
id: doc-012
title: Estudo 1 — execução Go sem CGO/libvips
type: other
created_date: '2026-11-12T23:39:00Z'
updated_date: '2026-11-12T23:39:00Z'
tags:
  - estudo
---
> Importado de `docs-dev/ESTUDO_01_GO_SEM_CGO.md`. Original removido em 12/11/2026; consulte o histórico do Git.

# Estudo 1 — execução Go sem CGO/libvips

Data: 06/11/2026. Estudo inicial do primeiro item do [roadmap](<../roadmaps/doc-001 - ROADMAP.md>).
Investigação específica posterior: [webp-go-pure, mudanças e prova isolada](<doc-013 - Substituição-de-libvips-por-webp-go-pure.md>).
Objetivo proposto: compilar e executar a aplicação sem CGO e sem bibliotecas
nativas de imagem instaladas, preservando o comportamento essencial. Isso não
significa eliminar módulos Go, fontes, certificados TLS, dados de fuso ou outras
dependências necessárias para compilar, operar e validar a aplicação.

## Diagnóstico reproduzido

Na árvore local e com Go 1.26.3 em macOS arm64, o comando
`CGO_ENABLED=0 GOPROXY=off go build -o /private/tmp/koito-estudo-sem-cgo ./cmd/api`
**falhou** em `github.com/h2non/bimg` por símbolos ausentes com CGO desligado.
Isso demonstra um bloqueador, não prova que ele seja o único; repetir o build
após a substituição para detectar os demais. Nenhum banco foi aberto.

O `go.mod` inclui `github.com/h2non/bimg v1.1.9`, `golang.org/x/image v0.33.0`
e `modernc.org/sqlite v1.49.1`. A aplicação usa `bimg` em:

1. `imagecache/imagecache.go`: checa dimensões das imagens baixadas, gera cópia
   source e cinco tamanhos (64, 128, 300, 640 e 1000), corta quadrado, retira
   metadados, converte para WebP e usa cache/singleflight/escrita atômica.
   Source tem lado máximo 1200 e usa qualidade 100; demais usam 85.
2. `internal/admin/artwork.go`: valida dimensões de upload/download antes de
   publicar capas. Avatares e outras imagens também chegam ao cache comum.
3. `Makefile`, `Dockerfile`, CI e documentação exigem CGO/libvips hoje. O
   `modernc.org/sqlite` é um driver distinto e não usa `bimg`.

O armazenamento existente usa `<tamanho>.webp`, MIME `image/webp` e URLs como
`/image/<id>/128x128.webp`. Não converter extensões em nomes de arquivos de
outro formato nem trocar silenciosamente o MIME. Preservar leitura do cache
antigo e verificar fallback/redownload quando `source.webp` não existir.

## Alternativas a comparar, sem decisão

| Opção | Vantagem pretendida | Pontos a provar |
| --- | --- | --- |
| Codec WebP e redimensionador em Go | Pode preservar URL, MIME e cache e dispensar biblioteca nativa | Compatibilidade com WebP antigo, codificação com/sem perdas, transparência, orientação EXIF, cores, formatos de entrada, qualidade, recursos e licença. |
| Decodificar WebP existente e passar a codificar JPEG/PNG | Pode usar mais recursos comuns do ecossistema Go | Exige nova convenção de cache/MIME e estratégia de transição; PNG pode aumentar arquivos e JPEG não preserva transparência. |
| Implementação compilada de C para WASM/Go | Pode manter compatibilidade sem CGO | Verificar se ainda usa biblioteca dinâmica, tag de build e dependências embutidas; medir tempo, memória, plataformas, manutenção e licença. |

O pacote oficial [`golang.org/x/image/webp`](https://pkg.go.dev/golang.org/x/image/webp)
documenta **decodificação**, não codificação; portanto não resolve sozinho a
geração de WebP. Como candidatos para investigação, e não recomendação aprovada,
[`webp-go-pure`](https://github.com/SeriousBug/webp-go-pure) declara codec Go sem
CGO, enquanto [`gen2brain/webp`](https://github.com/gen2brain/webp) declara fallback
transpilado e opção para não carregar biblioteca dinâmica. As afirmações são dos
projetos; não foram auditadas, instaladas, compiladas ou medidas aqui. Avaliar
licença, manutenção, versões, suporte a Go/plataformas e vulnerabilidades antes
de escolher.

Na revisão documental da versão `x/image v0.33.0`, identificaram-se avisos para
o decoder WebP anteriores a v0.42.0 e v0.43.0: [GO-2026-4961](https://pkg.go.dev/vuln/GO-2026-4961)
e [GO-2026-5061](https://pkg.go.dev/vuln/GO-2026-5061). O primeiro especifica
plataformas de 32 bits. `internal/summary/image.go` registra o decoder e contém
`addImage`, mas o gerador que o chamaria está comentado; não foi demonstrado
um caminho de exploração na aplicação atual. Antes de ativar decodificação Go,
verificar versões corrigidas, requisitos de Go e alcançabilidade com ferramenta
apropriada; não confundir versão afetada com exploração comprovada.

## Plano de prova e critérios de aceite propostos

1. Inventariar formatos realmente aceitos pela instalação atual e preparar
   corpus fictício JPEG, PNG, WebP, transparência, orientação EXIF, dimensões
   inválidas, arquivos truncados, metadados e limites de 10.000 por lado / 40
   milhões de pixels. Definir comportamento para GIF animado, AVIF e outros
   formatos antes de assumir que são suportados ou removidos.
2. Comparar saída com cópias da implementação atual: dimensões, enquadramento,
   orientação, transparência, metadados, MIME, legibilidade e tamanho. Medir
   latência e pico de memória, inclusive imagens grandes e requisições paralelas;
   estabelecer limites mensuráveis antes de escolher uma biblioteca.
3. Testar cache WebP v3 existente, geração sob demanda, redownload, falhas de
   disco, proteção contra excesso de bytes/pixels, concorrência, publicação
   atômica, cancelamento e limpeza sem usar imagens ou banco do usuário.
4. Compilar e testar `CGO_ENABLED=0` para as plataformas declaradas, executar o
   binário em ambiente sem libvips, verificar TLS e fusos, e testar DB SQLite v3,
   imagens, API ListenBrainz e administração. Medir tamanho e inicialização.
   Fazer imagem Docker e verificar instalação, configuração, upgrade e rollback.
5. Registrar a decisão de formato, biblioteca, limitações e benchmarks na issue;
   atualizar build/CI/Docker/docs apenas depois de a prova cumprir os critérios.

Este estudo não alterou código, dependências, banco, imagens ou configurações.
As alternativas e critérios são propostas para a próxima issue, não garantias de
resultado nem promessa de binário sem toda e qualquer dependência operacional.
