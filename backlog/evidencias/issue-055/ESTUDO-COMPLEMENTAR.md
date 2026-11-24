# ISSUE-055, 174 e 175 — complemento de estudo

Pesquisa preliminar de 24/11/2026. Complementa `ESTUDO.md`; a seção de decisões registra o que o usuário aprovou, e o restante contém investigação e propostas sem implementação.

## Decisões do usuário — 24/11/2026

- Fontes primárias: OpenSubsonic configurado e/ou ListenBrainz e MusicBrainz; Last.fm, Apple e Deezer como fallback opcional quando os dados faltarem, sujeitos à configuração e aos termos dos serviços. Não usar scraping.
- Separar tags populares de gêneros e incluir metadados de artistas. Preservar separadamente por perfil valores conflitantes de bibliotecas.
- Apresentação e gráficos ficam para a ISSUE-212. Enriquecer também o histórico gradualmente, com limites de requisições configuráveis e acompanhamento na central de monitoramento ligada à central de saúde (ISSUE-213).
- Um subsistema compartilhado, começando pelas ISSUE-174 e 175; a migração da 055 antecede a consolidação da ISSUE-195.

Os demais detalhes técnicos abaixo são propostas, não decisões aprovadas.

## Verificação do servidor

Em 24/11/2026, a consulta GET inicial retornou HTTP 403; um novo teste com POST de formulário autenticado e User-Agent de navegador funcionou. O `ping` informou Navidrome 0.64.2 e suporte OpenSubsonic. Não foi isolado se o bloqueio inicial dependia do método, do User-Agent ou de ambos. Não registrar credenciais ou URLs autenticadas em evidências, logs ou testes.

Na amostra aleatória de 100 músicas de `getRandomSongs`, 78 tinham `genres`, 98 `year`, 41 `musicBrainzId`, 26 `isrc` e 14 `bpm`. Cinco consultas `getSong` também responderam. Em outra amostra de 15 álbuns distintos, `getAlbum` retornou `releaseDate` e `originalReleaseDate` em 15, `genres` em 11, `recordLabels` em 11, `musicBrainzId` em 5 e `releaseTypes` em 5. As contagens indicam presença não vazia dos campos, não qualidade, precisão ou representatividade; amostras pequenas e aleatórias não são censo. A disponibilidade no servidor não significa que o scrobble envie esses dados ou seu ID nativo.

Outra amostra aleatória de 10 músicas foi consultada com `search3` pelo título, até 100 resultados, seguida de comparação sem distinção de caixa de título, artista e álbum: a música de origem apareceu nos resultados em 9 casos, com uma correspondência exata por esses três campos; um caso não retornou correspondência exata. Esta amostra não comprova unicidade global, ausência de truncamento ou correção automática e não substitui testes de duplicatas e versões.

## O que chega numa escuta

Conferência estática do [adaptador ListenBrainz Navidrome v0.64.2](https://github.com/navidrome/navidrome/blob/v0.64.2/adapters/listenbrainz/agent.go) e dos [tipos JSON](https://github.com/navidrome/navidrome/blob/v0.64.2/adapters/listenbrainz/client.go): uma escuta `single` inclui `listened_at`, `artist_name`, `track_name`, `release_name` e, quando disponíveis, `submission_client`, `submission_client_version`, `tracknumber`, `artist_names`, `artist_mbids`, `recording_mbid`, `release_mbid`, `release_group_mbid` e `duration_ms`. A notificação `playing_now` não inclui horário. Os campos opcionais podem ser omitidos. O adaptador não envia ID interno Subsonic, gêneros, tags, ano, datas, BPM, ISRC, número do disco ou artista do álbum. Isto é confirmado no código da versão, não uma captura de tráfego desta instalação; outros clientes podem enviar campos diferentes.

O TuneTrail decodifica esses campos e o snapshot seleciona nomes, duração, número de faixa e MBIDs. Embora aceite `additional_info.tags`, `ListenBrainzSnapshot` não as copia; ano e data não constam do tipo atual. Assim, para este Navidrome, gênero e ano exigem consultar o catálogo ou outra fonte. O ID de `getSong` não é o `recording_mbid`; para descobri-lo, `search3` devolve candidatos que precisam de confronto com artista, título, lançamento, duração, posição e MBIDs disponíveis. Divergência, mais de um candidato ou resultados potencialmente truncados não autorizam associação automática. Depois de uma associação segura, `getSong` e `getAlbum` fornecem os campos adicionais, guardando a origem por entidade.

## Correção sobre Last.fm

O estudo anterior não encontrou anúncio; isso ficou desatualizado. A [documentação de scrobbling do Navidrome](https://www.navidrome.org/docs/usage/features/scrobbling/) informa que, a partir de 1º de dezembro de 2026, o titular da chave de API deve ter Last.fm Pro. Isso afeta inclusive o planejamento do cliente de imagens existente quando depender de chave sem Pro; não demonstra encerramento total da API. Confirmar termos de armazenamento e uso antes de implementar um provedor de tags.

No TuneTrail, o cliente Last.fm atual é ativado quando há chave e é usado pela cadeia de imagens de artistas e álbuns (`internal/images/imagesrc.go` e `internal/images/lastfm.go`); há também importação de arquivo de histórico Last.fm (`internal/importer/lastfm.go`), independente de consulta de metadados pela API. Não há nessas rotas um provedor de gêneros para o catálogo.

## Contratos verificados

- [OpenSubsonic Child](https://opensubsonic.github.io/docs/responses/child/) prevê `genres`, `bpm`, `musicBrainzId` e `mediaType`; os campos são opcionais. `getSong` requer ID do servidor: um MBID ou nome de faixa não é automaticamente esse ID. A busca por texto precisa tratar versões e resultados ambíguos. A presença de `originalReleaseDate` na resposta de álbum deve ser confirmada separadamente; não consta no tipo Child consultado.
- [ListenBrainz metadata](https://listenbrainz.readthedocs.io/en/latest/users/api/metadata.html): `/1/metadata/recording/` parte de MBIDs e pode retornar tags de gravação, artista e grupo; `genre_mbid` distingue exemplos de gênero de outras tags. `/1/metadata/lookup/` recebe artista, gravação e lançamento opcional e exige token; retorno HTTP 200 não significa correspondência. Um resultado sugerido continua sendo declaração, não validação de identidade.
- [MusicBrainz API](https://musicbrainz.org/doc/MusicBrainz_API): `inc=genres` distingue gêneros das tags gerais; gravação, lançamento e grupo são entidades diferentes. A API pode limitar entidades associadas retornadas; não presumir que um resultado inicial contém todos os lançamentos. Respeitar [limites de requisição](https://musicbrainz.org/doc/MusicBrainz_API/Rate_Limiting).

## Confronto com o código

O cliente Subsonic atual em `internal/images/subsonic.go` destina-se a imagens e pesquisa por `search3`; não implementa o fluxo das issues 174/175. `internal/ingest/listenbrainz.go` monta um snapshot restrito e já inclui MBIDs de gravação, lançamento e grupo, inclusive mapeamentos; a persistência de tags e datas requer contrato explícito. A identidade separa `recordings` e `occurrences`, com ligação ao lançamento; não presumir que metadados de uma edição pertençam a todas as ocorrências da gravação.

## Proposta para implementação compartilhada

1. Criar contrato interno de enriquecimento com resultado tipado por entidade, campo, origem, instante e escopo; provedores só retornam afirmações, sem alterar identidade diretamente. Distinguir encontrado, ausente, ambíguo, desabilitado e erro transitório.
2. Admissão mantém ACK independente de rede. Agendador persistente ou recuperação demonstrável após reinício, deduplicação por entidade **e configuração/escopo do provedor**, limite por fonte, timeout, backoff e cache negativo separado de falhas transitórias. Não fazer rede sob lock de escrita. Desligar provedor não deve gerar retry infinito.
3. ISSUE-174: usar ID nativo apenas quando comprovadamente associado ao mesmo servidor; caso contrário buscar candidatos e exigir correspondência inequívoca. Registrar MBIDs e proveniência como declaração; não substituir MBID existente nem promover a validado sem o contrato vigente.
4. ISSUE-175: tentar lookup autenticado apenas quando configurado, com dados mínimos e token do contexto autorizado. Tratar versões e lançamentos semelhantes como ambíguos. O resultado não é prova; não sobrescrever uma declaração existente.
5. ISSUE-055: armazenar observações de gênero e tag multivaloradas com origem, entidade e escopo, preservando rótulo original; manter valores locais por perfil isolados até existir regra explícita de compartilhamento. Datas parciais devem preservar precisão e distinguir edição, primeiro lançamento do grupo e primeira publicação da gravação. Ausência ou erro não apaga dado anterior; edição manual prevalece conforme política de escopo.

Uma tabela genérica única não garante integridade referencial entre tipos de entidade: avaliar tabelas de vínculo por tipo ou uma camada de entidades com chaves reais, restrições e operações de mesclagem/exclusão testadas. Definir precedência por campo e escopo antes de materializar o valor selecionado. A migração aditiva deve anteceder a ISSUE-195 como já registrado.

## Possibilidades para gráficos

Ano original e da edição permitem décadas e distância entre lançamento e descoberta; gêneros permitem evolução do perfil e artistas explorados por gênero. Tipo de lançamento (álbum, EP, single, ao vivo) pode enriquecer rankings; BPM pode permitir faixas de ritmo quando houver cobertura. País de artista pode apoiar mapas, mas país de origem não é nacionalidade nem localização da escuta. Em todos os casos explicitar cobertura, múltiplos valores e ausência; não herdar gênero de artista como fato da gravação sem marcá-lo como inferência.

## Pendências antes de código

Obter respostas reais sanitizadas de `ping`, `getSong`, `getAlbum` e busca; verificar se a escuta transporta ID nativo e em qual namespace. Auditar os pontos atuais de uso do Last.fm, seus termos e limites. Definir política de dados privados e conflitos no catálogo compartilhado, retenção de fila, regras de correspondência, configuração por perfil e critérios de teste. Avaliar Advisor para concorrência e identidade conforme orientações do projeto.
