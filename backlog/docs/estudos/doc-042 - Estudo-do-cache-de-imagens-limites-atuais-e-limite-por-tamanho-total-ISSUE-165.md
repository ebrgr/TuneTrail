---
id: doc-042
title: >-
  Estudo do cache de imagens: limites atuais e limite por tamanho total
  (ISSUE-165)
type: other
created_date: '2026-11-17T17:34:00Z'
updated_date: '2026-11-17T17:34:00Z'
tags:
  - estudo
---
# Estudo do cache de imagens: limites atuais e limite por tamanho total

Origem: pedido do usuário em 2026-11-17 (ISSUE-165). Estudo estático do código em `imagecache/`, `internal/imagepipeline/`, `internal/imagefetch/`, `internal/catalog/images.go`, `engine/engine.go` e `internal/cfg/cfg.go`, com medição em uma cópia local de banco (sem dados pessoais nesta nota). Não houve alteração de código.

## 1. Como o TuneTrail guarda as imagens
- Pasta `image_cache/<2 primeiros caracteres do UUID>/<UUID>/` dentro do diretório de configuração. Cada imagem do catálogo (artista, álbum) ou avatar tem um UUID próprio.
- `source.webp`: a imagem original, normalizada para WebP com no máximo 1200 px no lado maior.
- Derivados `64x64`, `128x128`, `300x300`, `640x640` e `1000x1000`: gerados SOB DEMANDA a partir do `source.webp` (`GetImageContext`) e mantidos no disco para sempre.
- Escrita atômica (arquivo temporário e `rename`) e um portão de publicação (`LockPublication`) que coordena a gravação com a coleta de órfãs.
- O banco guarda o UUID (`image`) e a origem (`image_source`): uma URL para imagens remotas (Cover Art Archive, CDN de artistas) ou vazio para upload e avatar.

## 2. Limites que existem hoje
Todos são limites POR IMAGEM, não do cache:

| Limite | Valor | Onde |
| --- | --- | --- |
| Download remoto | 10 MiB e 10 s, com validação de redirecionamentos | `imagefetch.Download` |
| Arquivo aceito pelo pipeline | 10 MiB | `imagepipeline.MaxBytes` |
| Dimensão | 10000 px por lado e 40 milhões de pixels | `imagepipeline.MaxSide`, `MaxPixels` |
| Tamanho do original salvo | máx. 1200 px, em WebP | `SourceMaxSide`, `ImageSizeSource` |
| Derivados | 5 tamanhos fixos | `imagecache.go` |

## 3. O que NÃO existe
- **Nenhum limite do tamanho total** do cache, nem por quantidade de arquivos, nem por idade.
- **Nenhuma remoção por tamanho ou idade.** A única coleta é `PruneOrphanedImages`, executada uma vez por partida (`engine/engine.go:262`): apaga a pasta de UUIDs sem associação no banco.
- `KOITO_ENABLE_FULL_IMAGE_CACHE` é uma opção morta: está listada em `cfg` e em `env_alias.go`, mas nunca é lida.

## 4. Medição (cópia local de desenvolvimento)
- 36 imagens, 125 arquivos, cerca de 12 MiB no total: perto de 330 KiB por imagem com todos os tamanhos.
- `source.webp` de 160 a 270 KiB; derivados de 9 KiB (64 px) a poucas centenas de KiB (1000 px).
- Projeção: 1.000 imagens ≈ 330 MiB; 10.000 ≈ 3,3 GiB; 30.000 ≈ 10 GiB. O cache cresce em silêncio com o histórico, e o backfill de imagens a cada partida (ISSUE-117) o enche mais depressa.

## 5. Proposta: limite pelo tamanho total, apagando as mais antigas
Aprovada em princípio pelo usuário (ISSUE-165): trocar a ausência de limite por um teto de tamanho total; ao se aproximar do teto, apagar os arquivos mais antigos.

### 5.1 O que apagar primeiro (ordem)
1. **Derivados** (`64`, `128`, `300`, `640`, `1000`) mais antigos: são regeneráveis a custo baixo a partir do `source.webp`.
2. **`source.webp` de imagens remotas** (com `image_source` em URL), quando ainda faltar espaço: podem ser baixadas de novo.
3. **Nunca apagar:** avatares e uploads (sem URL de origem: não se recuperam), imagens com reserva de publicação ativa, e a imagem que está sendo servida na requisição.
- A unidade de remoção do passo 2 é a pasta inteira do UUID (derivados e original juntos), para não deixar derivado sem origem.

### 5.2 O que significa "mais antigo"
- Usar a **data de último acesso real** gravada pelo próprio aplicativo (por exemplo, `os.Chtimes` no `mtime` ao servir um arquivo, com intervalo mínimo de uma hora para evitar escrita a cada requisição). `atime` do sistema não é confiável (montagens `noatime`, contêineres).
- Alternativa mais simples (menos justa): `mtime` de criação. Apagaria uma capa muito vista só por ser antiga.

### 5.3 Quando limpar
- Verificação na gravação de um arquivo novo (soma incremental mantida em memória, recalculada a cada partida) e uma varredura periódica.
- **Marca d'água com histerese:** ao passar de 100% do teto, apagar até 90%, para não limpar a cada imagem nova.
- A coleta deve usar o mesmo portão de publicação (`LockPublication`) da coleta de órfãs, para não concorrer com uma gravação em andamento.

### 5.4 Consistência com o banco
- Apagar só derivados é invisível: a próxima requisição regenera.
- Apagar um original remoto exige que a próxima requisição **baixe de novo** de `image_source` (rebaixar sob demanda, com o limite de 10 MiB e 10 s) ou, enquanto isso, mostre o ícone padrão. Hoje, sem o original, `GetImage` devolve erro e a página mostra o marcador de imagem.
- Evitar um laço: baixar a imagem de novo e apagá-la logo em seguida se o teto estiver pequeno demais. Impor um piso para o teto (por exemplo, 64 MiB) e nunca remover o que foi acessado na última hora.

### 5.5 Configuração
- Teto em MiB, por exemplo `image_cache.max_mb` no `conf.toml` (coordenar com a ISSUE-153) e variável de ambiente `KOITO_IMAGE_CACHE_MAX_MB`; `0` ou ausente significa sem limite.
- Valor padrão sugerido: **1024 MiB** (cabe cerca de 3.000 imagens com todos os tamanhos). O usuário decide.
- Remover a opção morta `KOITO_ENABLE_FULL_IMAGE_CACHE` ou documentá-la como obsoleta.

### 5.6 Observabilidade
- Registrar em log o tamanho atual, o teto e o total apagado em cada limpeza; expor o uso do cache na administração (ISSUE-112/127).

## 6. Riscos
- Avatares e uploads perdidos por engano (ver 5.1; teste obrigatório).
- Baixar de novo uma imagem cuja fonte mudou ou saiu do ar: ficaria sem capa. Mitigar com o ícone padrão e a nova busca (ISSUE-117).
- Corrida entre limpeza e gravação, e entre limpeza e a coleta de órfãs.
- Disco cheio antes do teto (teto maior que o espaço livre): avisar na administração.
- Custo da varredura em caches grandes: manter soma incremental em vez de percorrer a pasta a cada imagem.

## 7. Perguntas ao usuário
1. Teto padrão: 1 GiB (recomendado), ou sem limite até o usuário configurar?
2. "Mais antigo" por último acesso (recomendado) ou por data de criação?
3. Apagar também originais remotos quando faltar espaço (recomendado, com rebaixa sob demanda), ou apagar só derivados e avisar quando só restarem originais?
4. Configurar só por `conf.toml` ou também por variável de ambiente?

## 8. Relação com outras issues
ISSUE-117 (estudo de imagens: backfill, nova busca, itens sem imagem), ISSUE-153 e 127 (configuração), ISSUE-112 (logs e administração), ISSUE-102 (contêiner e volumes), ISSUE-103 (backup).

## 9. Decisões do usuário (2026-11-17)
1. **Teto padrão, configurável pelo administrador root** (valor padrão sugerido: 1024 MiB; a configuração é feita por quem tem o papel de admin).
2. **"Mais antigo" = por último acesso**, gravado pelo aplicativo.
3. **Aceita a orientação do estudo:** apagar primeiro os derivados e depois os originais remotos mais antigos, com nova baixa sob demanda; nunca avatares, uploads nem imagens com publicação em andamento.
4. **Configuração no `conf.toml`** (sem variável de ambiente nova), reaproveitando o mecanismo da ISSUE-153.
