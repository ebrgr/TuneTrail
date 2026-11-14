---
id: doc-034
title: Estudo da ISSUE-114 — issues e PRs abertos do Koito
type: other
created_date: '2026-11-14T23:00:00Z'
updated_date: '2026-11-14T23:00:00Z'
tags:
  - estudo
---
**Tarefa:** ISSUE-114 · **Base:** `main` em `1cf5535`, TuneTrail 0.13 · **Data:** 14/11/2026 · **Fonte:** [gabehf/Koito](https://github.com/gabehf/Koito), última versão v0.3.2.

Estudo sem mudança no produto. O levantamento cobre as **79 issues** e os **21 pull requests** abertos no Koito em 14/11/2026, incluindo o roadmap publicado na issue #284. A lista coletada está em `backlog/evidencias/issue-114/koito-abertos-2026-11-14.json`.

O TuneTrail partiu do Koito 0.3.2 e trocou o cliente React por páginas Go. Por isso, os problemas da interface React (SPA) não se aplicam, mas os do servidor, do catálogo e dos importadores podem ter vindo junto. Os itens marcados **confirmado** foram verificados no código atual. Os marcados **verificar** ainda precisam de reprodução.

## Resumo

| Grupo | Itens do Koito | Destino |
|---|---:|---|
| Já resolvidos no TuneTrail | 9 | Nada a fazer |
| Bugs confirmados no nosso código | 5 | ISSUE-115 (4) e ISSUE-116 (1) |
| Cobertos por tarefas existentes | 34 | Comentário ou ajuste de escopo nas tarefas |
| Novos estudos | 6 tarefas | ISSUE-116 a ISSUE-121 |
| Não se aplicam | 10 | Registrados para não serem reavaliados |

## Roadmap publicado no Koito (#284)

| Item do Koito | Situação no TuneTrail | Destino |
|---|---|---|
| 0.4.0 Refatoração: adaptadores externos, pacotes catalog e db | Estrutura própria já divergiu (store SQLite, catálogo e adaptadores próprios) | Não acompanhar |
| 0.4.0 Guardar tags na submissão e importação (#225) | Não existe | ISSUE-055 |
| 0.4.0 Uma faixa em vários lançamentos (#34) | Não existe; o dono do Koito considera não trivial | ISSUE-116 |
| 0.4.0 Escopos em chaves de API; rotas só de sessão | Chaves por perfil existem, sem escopos | ISSUE-121 |
| 0.5.0 Reescrita da configuração (#285) | Feita aqui: TOML, `TUNETRAIL_*` e página de configurações | Resolvido |
| 0.5.0 Apagar todas as imagens (#186) | Não existe | ISSUE-117 |
| 0.5.0 Spotify como fonte de imagens (#303) | Não existe; a API do Spotify restringiu o modo de desenvolvimento em 2026 | ISSUE-117 |
| Comentário: álbum com artistas corretos (discussão #271) | Corrigido na ISSUE-049 (0.9) | Resolvido |
| Comentário: sincronizar favoritos do Jellyfin/Plex | Não existe | ISSUE-118 |

## Já resolvidos no TuneTrail

| Koito | Assunto | Onde |
|---|---|---|
| #11, PR #374 | Vários usuários na mesma instância | Perfis de escuta (SUM); o PR #374 é a origem do TuneTrail |
| PR #219 | `listen_type` na chave de coalescência | `lbz_submit_listen.go` já inclui o tipo na chave |
| #82, #196, #330 | Apagar uma escuta só | Botão por linha, DELETE com status e sem repetição (ISSUE-106) |
| #113 | Temas com origem no servidor | `themes.css` e `palettes.json` gerados no servidor |
| #197 | Tema personalizado não persiste | `theme.js` relê as preferências e restaura o tema |
| #152, parte do PR #300 | Início da semana configurável | Preferência por perfil (`data-week-start`) |
| #285 | Reescrita da configuração | TOML e página de configurações |
| #309 | Nomes longos sem espaço vazam do card | `overflow-wrap: anywhere` e corte em 3 linhas |
| #364 | `curl` na imagem para o healthcheck | Imagem já inclui `curl` |

## Bugs confirmados no nosso código

| Koito | Problema | Evidência no TuneTrail | Destino |
|---|---|---|---|
| #74 | Endereço IPv6 no bind falha | `cfg.ListenAddr` monta `%s:%d` sem colchetes | ISSUE-115 |
| PR #373 | `listened_at` decimal recusado | `listenbrainz.Listen.ListenedAt` é `int64` | ISSUE-115 |
| PR #349, #211 | Ping Subsonic com `v=1` falha no LMS | `subsonic_ping.go` envia `v=1` | ISSUE-115 |
| #185 | Mover a importação entre volumes falha | `os.Rename` em `importer.go` e `profile_import.go` | ISSUE-115 |
| #369 | Alias secundário ignorado ao associar por título | `trackByTrackInfo` consulta só `tracks_with_title` | ISSUE-116 |

## Cobertos por tarefas existentes

| Koito | Assunto | Tarefa |
|---|---|---|
| #16, #324, #362, #368 | Rankings e gráficos por tempo ouvido | ISSUE-092, ISSUE-097 |
| #65, #325, #354 | Duração das faixas e `duration_played` | ISSUE-097 |
| #295 | Comparação com o período anterior | ISSUE-093 |
| #78 | Grade de capas para compartilhar | ISSUE-098, ISSUE-099 |
| #103 | Endpoint compatível com Last.fm | ISSUE-091 |
| #110 (parte), #366 | API ListenBrainz e exportação por API | ISSUE-087 |
| #225 | Gêneros e tags | ISSUE-055 |
| #304 | Clicar no dia do heatmap para ver as escutas | ISSUE-072 |
| #319 | Última escuta como coluna ao lado das faixas | ISSUE-072 (seguir a ISSUE-113) |
| #363 | Destaque de "tocando agora" | ISSUE-072, ISSUE-084 |
| #314 | Pular várias páginas na paginação | ISSUE-073 |
| #6, #220, #338, #340 | Fusão: várias seleções, identificação, busca e faixas do álbum | ISSUE-108 (e ISSUE-116 para as regras) |
| #200 | Fonte sem suporte a vietnamita | ISSUE-070 (avaliar subconjunto Noto) |
| #365 | Fonte grande do Android quebra o layout | ISSUE-069, ISSUE-078 |
| #310 | Legibilidade do título longo sobre a capa | ISSUE-073, ISSUE-078 |
| #350 | Relay recebe 429 do ListenBrainz | ISSUE-082 (outbox e retentativa) |
| #356 | Backup do SQLite | ISSUE-103 |
| #307 | Tamanho da imagem Docker | ISSUE-102 |
| #327, PR #329 | Jellyfin: títulos e capas | ISSUE-089 (imagens na ISSUE-117) |
| PR #305 | Eixos e tooltip no gráfico de interesse | ISSUE-077 |
| #269, #371, PR #342 | Unidades de tempo e separador de milhar | Interface multilíngue; incluir na ISSUE-073 |
| #63 | Personalizar título e descrição da página | ISSUE-074 (configurações) |

## Novos estudos

| Tarefa | Itens do Koito | Por que estudar antes |
|---|---|---|
| **ISSUE-116** — identidade de faixas | #34, #217, #214, #369, #359, #360, PR #315 | Muda o modelo de faixa e lançamento e exige migração |
| **ISSUE-117** — imagens | #55, #186, #298, #303, #306, #321, #333, #344, #377, PR #329, #336, #370 | Fontes, custo do backfill a cada partida (**confirmado** em `engine.go`) e configuração |
| **ISSUE-118** — favoritos e feedback | #104, #110, #312, PR #328 | Modelo novo por perfil e endpoints compatíveis |
| **ISSUE-119** — importadores | #205, #208, #210, #221, #375, PR #323, #334, #335, #376, #378 | Formatos novos, erros silenciosos e velocidade; vários itens a **verificar** |
| **ISSUE-120** — períodos e exclusões | #69, #346, PR #332, #347, #358 | Afeta a semântica das estatísticas (ISSUE-092) |
| **ISSUE-121** — implantação | #143, #339, roadmap #284 (escopos) | Afeta cookies, CSRF, links e perfis públicos |

## Não se aplicam

| Koito | Motivo |
|---|---|
| #301, #343 | Erros do cliente React/SPA, que o TuneTrail não usa |
| #203 | `LOGIN_GATE` do cliente React; o TuneTrail tem perfis públicos e administração separada |
| #215 | Atualização de binário a partir do Koito 0.1.4; o TuneTrail tem migração e recuperação próprias (ISSUE-064) |
| #308 | Apple Container; a imagem OCI já funciona, só documentar se pedirem |
| #222 | Limite de tamanho do próprio ListenBrainz no cliente; verificar só se aparecer no TuneTrail |
| #341 | Alias do MusicBrainz com MusicBrainz desligado: com `DISABLE_MUSICBRAINZ` o TuneTrail não cria o cliente (`engine.go`); **verificar** com o multi-scrobbler antes de descartar |
| PR #300 (formatos) | Datas e relógio já seguem o idioma e o fuso do perfil |
| PR #345 | Documentação do Koito |
| #284 (refatoração 0.4.0) | Estrutura interna já divergiu |

## Recomendação para o roadmap 1.0

1. **ISSUE-115** entra na Fase 2 (`m-11`) logo no início: são quatro correções pequenas, com teste, sem migração. Publica versão.
2. **ISSUE-116** e **ISSUE-117** são os estudos de maior valor: ambos tratam de problemas recorrentes no Koito (identidade de faixas e imagens) e devem terminar antes da ISSUE-081 (deduplicação) e da Fase 6, respectivamente.
3. **ISSUE-118** a **ISSUE-121** ficam como estudos nas fases indicadas; viram tarefas de implementação só depois do aceite de cada estudo.
4. Os itens cobertos por tarefas existentes não mudam a ordem do roadmap. Ao iniciar cada tarefa, consultar a tabela acima e citar a issue do Koito como referência.
5. Repetir este levantamento antes da candidatura à 1.0 (ISSUE-105), comparando com `koito-abertos-2026-11-14.json`.
