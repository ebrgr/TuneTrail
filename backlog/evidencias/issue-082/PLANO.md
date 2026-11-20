# ISSUE-082 — Outbox durável para o relay ListenBrainz

Versão 2, fase E0. Base `v0.47` (`a178347`), SQLite v11. Parecer de Opus (esforço high): **REPROVADO COM RESSALVAS**. O usuário aprovou as recomendações em 20/11/2026. A implementação aguarda a revisão curta do contrato emendado. A tabela de payload segue para validação final.

## Caminho atual na v0.47

| Ponto | Comportamento observado na base |
| --- | --- |
| `engine/handlers/lbz_submit_listen.go:154–168,194–235,387–416` | Lê os bytes antes do decode. Limites de 16 MiB e 1.000 itens por pedido, com validação de 10 KiB por item e JSON residual na via ativa. Token de entrada identifica o perfil. Para histórico ativo, admissão de lote faz commit dos recibos antes de escrever o 200. Logo depois consulta o relay e dispara o corpo integral, mesmo para reenvios ou itens que só serão classificados depois. |
| `engine/handlers/lbz_submit_listen.go:241–376` | Na via legada, inclusive `playing_now`, a consulta da configuração e o disparo da goroutine precedem a escrita do 200. Na via ativa vêm depois de `writeListenOK`. Em ambas, espera-se apenas pela consulta local, e o cliente recebe `200`, `Content-Type: application/json` e `{"status": "ok"}` sem esperar pelo HTTP externo. `doLbzRelay` usa contexto de fundo e registra apenas o aviso `ListenBrainz relay submission failed` ao esgotar ou falhar. Configuração ausente, incompleta ou erro de leitura não dispara envio nem aviso nesse ponto. |
| `internal/relay/relay.go:23–65,91–139` | POST para base normalizada + `/submit-listens` (preserva prefixo), corpo recebido byte a byte, `Authorization: Token <token do destino>`, `Accept: application/json`, `Content-Type: application/json`. Sem redirects. HTTP(S) autogerenciado permitido. Aceita 2xx com resposta lida dentro de 64 KiB. Timeout de 10 s por requisição, janela de 3 min; repete apenas 5xx com 5 s iniciais, dobrando até 40 s. 429, outros <500 e falhas de transporte/leitura encerram a tentativa atual, sem interpretar `Retry-After`. |
| `db/migrations_sqlite/003_listening_profiles.sql:86–94`, `internal/db/sqlite/profile_store.go:111–166`, `engine/handlers/profile_relay.go:37–95` | URL e token ficam em `profile_relays`, em texto no SQLite, por `profile_id`. O GET administrativo retorna estado, URL e token com `Cache-Control: no-store` (`profile_relay.go:39–46`). A administração salva/remove a configuração e testa a credencial via `/validate-token`. No boot (`engine/engine.go:174–178`, `internal/cfg/cfg.go:30–32,215–218`), configuração histórica `KOITO_*` com aliases `TUNETRAIL_*` pode ser associada uma vez. `legacy_relay_binding` impede reassociação automática após revogação/exclusão. O token de entrada e o token de saída têm papéis distintos. |
| `db/migrations_sqlite/004_ingest_infrastructure.sql:8–19` e `008_recording_identity.sql:227–276` no mesmo diretório | `integrations` guarda identidade, tipo, referência opcional de segredo, revogação e retenção por perfil. A 008 religa `outbox` a listens pelo par perfil/evento, com cascata, e integrações pelo par perfil/destino. Há unicidade por perfil/destino/evento/operação e por perfil/destino/chave. Índices da fila aberta e do evento, lease com versão e detalhes JSON separados. A 008 ainda bloqueia INSERT da outbox. |
| `db/migrations_sqlite/009_ingest_activation.sql:18–68` | Libera revisão ativa e INSERT de outbox só com recibo ativo `accepted` do mesmo perfil/evento. Sombra não materializa decisões. A guarda permite concluir trabalho admitido antes de desligar a admissão. |
| `db/migrations_sqlite/010_occurrence_ingest.sql:2–8,71–105,185–229` e `011_occurrence_writer_guards.sql:4–98` | Fixam a variante do recibo e a flag unidirecional. Protegem origem, vínculo com catálogo e versão da decisão. A 011 exige drenagem da fila v9, limpeza da sombra sem vínculo e ausência de relay para converter, bloqueia configurar relay após o corte e protege release vinculado. |
| `internal/db/sqlite/ingest_resolve.go:211–270` e `ingest_occurrence.go:177–255` | A 081 grava evento, decisão e revisão juntos. Para `accepted` vindo da API ListenBrainz com relay configurado, cria `relay:<profile_id>` do tipo `listenbrainz-relay` e intenção `discarded`, com `closed_at`, chave igual ao evento e motivo `pre_082_best_effort`. A marca não prova tentativa nem sucesso remoto. A 134 grava decisões por ocorrência sem intenção, pois o perfil convertido não pode ter relay. |
| `engine/ingest_recovery.go:15–47`, `engine/engine.go:249–252,280–291`, `internal/db/sqlite/ingest_resolve.go:88–110,152–172,237–251,273–299` | A recuperação roda mesmo com a admissão desligada, pesquisa sem reservar o escritor quando ociosa, faz claim via UPDATE condicional, lease de 120 s e incremento de work_version/attempt_count. Tick de 1 s, rajada de até 32. Valida versão, dono e lease antes de concluir. Na 134 a prova externa tem timeout de 20 s fora do lock. Reusar a disciplina de claim/CAS e lifecycle, com worker e estado próprios para entrega. |

Doc-010 §4.2 pede a troca do transporte por um destino da outbox. Doc-017 (020/034) registra a lacuna em 429, durabilidade e shutdown. Doc-039 fixa o evento como identidade da escuta. Doc-040 §§6–9 e decision-009 orientam elegibilidade, exclusão, privacidade, retenção e pelo menos uma vez. C6 da 081 mediu recuperação com lease real e retry persistente do resolvedor, tratado pela 173. As ressalvas da 081 e o ensaio E4 da 134 orientam os testes de CAS, rollback, replay e contenção. As consultas de ai-memory para ISSUE-082, outbox e relay apontaram os planos da 081/133 e o histórico de credenciais por perfil. Os fatos acima foram conferidos na worktree.

## Admissão e elegibilidade

`AdmitIngest` (`internal/db/sqlite/ingest_receipts.go:99`) fixa e devolve a política de entrega na transação que escolhe o modo. Todo recibo ativo novo na 082 nasce `durable`, sem interruptor próprio. Recibos existentes mantêm `legacy`. Um replay conserva os recibos e suas políticas originais. Para pedidos na via ativa da 082, inclusive replay de recibos antigos e lotes com recibos de ambas as políticas, a política efetiva do pedido é durável e o handler suprime o POST bruto. Assim, um replay ativo não torna elegível um recibo antigo nem reencaminha um lote inteiro. O handler só chama o relay bruto na via legada/sombra ou para `playing_now`. A via legada conserva a consulta da configuração, o corpo, os cabeçalhos, a URL, a resposta e o sender da v0.47.

O ACK ativo continua sendo o 200 após commit dos recibos, com o mesmo envelope. A fila de entrega nunca condiciona o ACK. Mantêm-se as validações de entrada, incluindo 10 KiB por item, e o 503 com `Retry-After: 1` se a admissão durável estiver indisponível no SQLite. A outbox não reserva capacidade nem valida envelope antes do ACK. O resolvedor roda após o commit de admissão. Ao confirmar a decisão, a intenção já está elegível. O servidor não estabelece barreira de ACK em memória ou persistida e não consegue confirmar que o cliente recebeu o 200.

Nas duas variantes, v9 e occurrence, a decisão de histórico da API ListenBrainz consulta o destino na mesma transação. Só um novo evento `accepted` com política durável e relay configurado cria intenção elegível. Inserir depois do UPDATE para `accepted` para satisfazer `trg_outbox_active_insert` (`009_ingest_activation.sql:59–68`). Evento, decisão, revisão e intenção têm o mesmo commit. `pending_identity`, `pending_duplicate`, `duplicate`, `invalid` e falha do resolvedor não enviam. Revisão futura que aceite evento novo aplica a mesma regra. Arquivos e fontes de sincronização ficam inelegíveis. Um recibo v9 legado em voo no upgrade conserva a regra do marcador `discarded/pre_082_best_effort` quando aplicável. Marcadores existentes continuam fechados e envios brutos interrompidos pelo upgrade podem se perder.

## Contrato de payload para validação final

Formatar deterministicamente a partir de `original_snapshot` imutável (`008_recording_identity.sql:192`), `ingest_events.listened_at` e `outbox.format_version`. A versão fixa o mapeamento, a representação e a ordem de campos. Conservar o encoder das versões ainda recuperáveis. Nenhuma mudança em `Snapshot` ou `Encode`, que definem os digests e a idempotência atuais. A 012 não acrescenta tabela ou cópia de payload, e a purga de detalhes nunca remove a fonte de reconstrução. Testar a permanência e unicidade da origem `accepted` durante revisão, merge e exclusão. Gerar bytes fora da reserva de escritor e validar a mesma saída após restart, retry e bisseção.

| Campo no recibo | Campo no item enviado v1 | Regra e diferença observável |
| --- | --- | --- |
| `listened_at` do recibo | `listened_at` | Epoch inteiro em segundos. Ausente/zero na entrada usa instante admitido, que antes podia sair ausente ou zero. Frações ficam no recibo, sem extensão no envio. |
| `Artist`, `Title`, `Release` | `track_metadata.artist_name`, `track_name`, `release_name` | Campos admitidos. Omitir álbum vazio. |
| `ArtistNames` | `additional_info.artist_names` | Conservar ordem e omitir lista vazia. |
| `DurationMS` | `additional_info.duration_ms` | Valor positivo. `duration` de entrada, quando presente e não zero, já prevaleceu na captura e virou ms. |
| `Client` | `additional_info.submission_client` | Campo unificado. Quando havia `media_player`, ele prevaleceu na captura. Omitir valor vazio e não inventar um `media_player` separado. |
| `ClientVersion` | `additional_info.submission_client_version` | Omitir na v1. A versão capturada pertence ao cliente de submissão original, cuja identidade pode ter sido substituída pelo player (`internal/ingest/listenbrainz.go:14–16,36–38`). |
| `DiscNumber`, `TrackNumber` | `additional_info.discnumber`, `tracknumber` | Emitir inteiro positivo. Strings inteiras válidas já foram convertidas. Valores <1 ou não inteiros não sobrevivem (`listenbrainz.go:65–72`). |
| `TrackMBID`, `Recording`, `ReleaseMBID`, `ReleaseGroupMBID` | `additional_info.track_mbid`, `recording_mbid`, `release_mbid`, `release_group_mbid` | UUIDs válidos e não nulos. Manter o significado de cada um. |
| `ArtistMBIDs` | `additional_info.artist_mbids` | UUIDs válidos e não nulos, na ordem disponível. |
| `MappedArtistMBIDs`, `MappedRecording`, `MappedReleaseMBID`, `ArtistMbidMappings` | `mbid_mapping.artist_mbids`, `recording_mbid`, `release_mbid`, `artists[]` | Preservar a distinção entre MBIDs declarados e mapeados. Pares de crédito usam `artist_name` e `artist_mbid`. Omitir grupos vazios, validar interoperabilidade no ensaio. |
| `MBIDSource`, `MBIDDeclaredAt` | Omitidos | Proveniência retida no recibo. Não promover identificação inferida a identificador de entrada. |
| `OriginalTime`, `TimeSemantics`, `CatalogTrackID`, `SkipCacheImage` | Omitidos | Precisão/evidência local e controles internos. |
| Extensões, URLs, headers, tokens e demais campos ausentes do snapshot | Omitidos | Sem consulta ao catálogo mutável e sem captura bruta. Omitir objetos opcionais vazios. |

Fixtures-base para validar a tabela com clientes reais, usando metadados de teste e sem credenciais:

| Origem da fixture | Entrada representativa | Saída esperada |
| --- | --- | --- |
| [ListenBrainz oficial, exemplo de submissão](https://listenbrainz.readthedocs.io/en/latest/users/api/core.html#submit-listens) | `single` com `listened_at=1780000000`, artista A, faixa B, álbum C. Variante de borda com horário ausente. | Mesma tripla e timestamp em segundos. Na variante, timestamp da admissão. |
| [multi-scrobbler, exemplo de POST do cliente](https://github.com/FoxxMD/multi-scrobbler/discussions/387) | `single` com `submission_client=multi-scrobbler`, versão, `artist_names`, `artist_mbids=[""]` e créditos de álbum suplementares. Variante com `media_player` e posição inválida. | Cliente e nomes disponíveis, sem versão, MBID vazio ou créditos não capturados. Na variante, cliente unificado recebe o player e omite posição inválida. |
| [Navidrome, fixture do POST de scrobble](https://github.com/navidrome/navidrome/blob/master/tests/fixtures/listenbrainz.scrobble.request.json) | `single` com `tracknumber=1`, `duration_ms=142200`, artistas múltiplos e MBIDs. Para Symfonium via Navidrome, a fixture de saída é do servidor Navidrome. | Posição, duração e artistas preservados. Os placeholders não UUID da fixture são omitidos. Variante com UUIDs válidos comprova sua preservação. |

São casos propostos com base em exemplos e fixtures públicas. E4 deve conferir a saída com versões identificadas e reproduzir Symfonium via Navidrome com biblioteca sintética. Adicionar casos locais de timestamp fracionário, duração concorrente, posições string/número e MBIDs mapeados. Comparar bytes do item por evento, inclusive após trocar a composição do micro-lote.

## Outbox, estado do destino e recuperação

Usar `relay:<profile_id>` como identidade estável em `integrations`, com `operation=listen` e `delivery_key=event_id`, como na 081. Unicidade e FKs compostas de perfil e destino/evento da 008 protegem isolamento. A geração é atributo mutável e cerca claims, sem integrar a chave. Consultar configuração na decisão, sem confirmação de geração na admissão. Trigger em `profile_relays` cobre INSERT/UPDATE/DELETE, inclusive `InitializeLegacyRelay`. Mudança de URL ou remoção invalida a geração, limpa a pausa anterior e passa pendências antigas para `failed/destination_changed`, preservando chave e origem do payload. Reprocessamento para URL diferente é ação explícita do operador. Troca só do token preserva linhas e chaves. Mudança efetiva de configuração libera pausa por credencial/rota. Uma revisão da configuração impede que uma resposta de credencial antiga volte a pausar o destino após a rotação. A troca de configuração não libera a posse física do lote em voo antes de seu término ou da expiração do lease, preservando o limite de um envio por destino. Criação de intenção, mudança de configuração e claim disputam a mesma disciplina de escrita.

| Estado do item | Transição |
| --- | --- |
| `pending` | Inserido após decisão elegível. Admitir até 100 mil linhas elegíveis abertas por destino, incluindo claims. Ao alcançar o teto, novas intenções nascem `failed/queue_capacity`, recuperáveis quando houver espaço. O limite controla trabalho de fila e admite linhas de falha retidas acima dele. |
| Enviando | Claim usa `pending`/`retryable`, `lease_owner`, `lease_until` e incremento de `work_version`, sem gastar tentativa do item pelo simples claim. Lease do destino impede segundo lote simultâneo, inclusive entre processos. |
| `retryable` | Trabalho mantido após liberação recuperável, inclusive cancelamento. Se houver pausa, a elegibilidade depende do relógio do destino. |
| `sent` | HTTP 2xx. Limpar lease, gravar `closed_at`, última tentativa e resultado mínimo. |
| `failed` | `queue_capacity`, `destination_changed`, `expired` aos 30 dias, ou falha do próprio item. Sem lease/agendamento, com reconstrução preservada até ação. |
| `discarded` | Descarte explícito ou marcador antigo já encerrado. Exclusão local prevalece e remove outbox/chave por cascata. |

Cada destino tem estado persistido independente dos itens, com `paused_until`, motivo, contagem de falhas do destino e posse/versão do lote. Um único POST em voo por destino, concorrência global limitada, rodízio entre destinos e nenhuma ordem garantida entre escutas. Selecionar até 100 itens elegíveis da mesma geração. `listen_type=import` com mais de um e `single` com um. Bytes do item permanecem estáveis. Claim por UPDATE condicional dentro de transação curta quando houver múltiplas linhas e dependências. Usar o padrão `BEGIN IMMEDIATE` e evitar reserva se não houver trabalho. HTTP e DNS ficam fora da reserva. CAS de conclusão exige perfil, work_version, lease_owner e `lease_until>now`, com geração e posse do lote válidas. Lease bem acima do timeout HTTP e da margem de persistência. Revalidar antes de enviar e descartar conclusão obsoleta. Revogação/exclusão concorrente pode coincidir com efeito em voo, sem recriação local.

Em 400/413, dividir o lote rejeitado em duas partes e repetir até isolar o item ofensivo. Só o item isolado consome tentativas, até oito, com agendamento quando recuperável e `failed` definitivo para rejeição determinística. Validar pelo ensaio o pressuposto de rejeição integral em 400/413. Registrar 2xx dos sublotes antes de prosseguir, revalidar lease e destino antes de cada POST e reagendar subconjuntos restantes se tempo, rate limit ou shutdown interromperem a bisseção. Fixar em E2 um orçamento de POSTs por claim para limitar o custo de isolar muitos itens ruins.

429, 5xx, timeout e conexão recusada pausam o destino, preservando o estado lógico dos itens e sem consumir tentativas por item. 401/403 e 3xx pausam até mudança de configuração. Combinar backoff exponencial de base 30 s, fator 2 e teto de 1 h com full jitter. Respeitar o maior entre o atraso local e o prazo válido de `Retry-After` (segundos ou data HTTP) ou `X-RateLimit-Reset-In`. `X-RateLimit-Remaining=0` provoca pausa preventiva, inclusive após 2xx, com prazo do servidor ou atraso local quando faltante. Tratar overflow, valores inválidos e passados. O teto de 1 h limita o atraso local. `Retry-After` longo mantém o destino pausado, e itens podem expirar independentemente, inclusive sob pausa. A idade máxima de 30 dias leva a `failed/expired` com reconstrução preservada. Coordenar recuperação explícita e renovação de janela com a 083.

Um único helper de backoff com relógio injetável, teto e full jitter atende 082 e 173 com parâmetros próprios por domínio. A 082 não altera `retryIngest` (`internal/db/sqlite/ingest_resolve.go:152–158`). A 173 adota o helper para o resolvedor, inclusive sua política terminal e a R-b. O agendador da 086 pode seguir as mesmas convenções. A 087 faz prévia sem efeito e importa sem relay automático. A 088 admite histórico pelo pipeline, com `getNowPlaying` efêmero. Checkpoint da fonte depende de admissão local durável.

### Transporte, diagnóstico e retenção

A tentativa durável retorna status, classe e espera do servidor. HTTP 2xx significa sucesso assim que os cabeçalhos válidos chegam, mesmo se a leitura do corpo falhar ou exceder 64 KiB. Descartar corpo de modo limitado e fechar. Preservar `Sender.Send` e sua semântica legada (`internal/relay/relay.go:61–63,98–139`). Manter URL normalizada com prefixo, POST, `Authorization: Token`, JSON, limite de tempo por requisição, redirects desligados e verificação TLS.

Dono não-root pode configurar a URL (`internal/admin/profiles.go:65–80`, rotas de perfil em `routes.go:80–83`). O transporte padrão atual usa `ProxyFromEnvironment` quando `Transport` é nil. Para entrega durável e `/validate-token`, usar dialer próprio que resolve DNS, classifica endereços e disca diretamente o IP permitido, preservando hostname para SNI/TLS. Bloquear loopback, privados, link-local, metadados e endereços IPv6 mapeados que contornem a regra, normalizando antes da classificação. HTTP e rede privada só com lista de permissão do operador root. Proibir destinos de metadados e demais exceções inseguras. Proxy desligado por padrão na via nova, com opção explícita do operador sujeita às mesmas garantias. Preservar o sender legado durante o replay de paridade.

Tokens ficam em `profile_relays`. Nenhum token em fila, payload, logs, evidência ou exportação. Logs sanitizados e limitados em frequência. Propor mascarar o token atualmente devolvido pelo GET administrativo e permitir atualizar URL sem revelar ou sobrescrever inadvertidamente a credencial. A 083 consulta por perfil/destino ou par perfil/evento, com estado do item, estado/motivo/prazo da pausa, tentativas, última tentativa, próxima elegibilidade, fechamento e motivo sanitizado. Agregar fila e último sucesso, excluindo marcadores antigos de sucesso/falha. Quem não é root vê só a classe HTTP. Root pode ver código numérico sanitizado. Texto, visibilidade e reprocessamento/descarte em coordenação com a 083.

Falhas ficam indefinidamente até ação do operador. Linhas `sent` ficam enquanto o evento existir. Detalhes opcionais fechados seguem 90 dias por padrão, entre 7 e 3.650 dias. A purga permanece explícita (`internal/db/sqlite/ingest_retention.go`) e não toca a reconstrução. Exclusão remove a chave local sem tombstone. Entrega pelo menos uma vez admite duplicação quando há aceite remoto sem CAS. Verificar a tendência do ListenBrainz oficial de descartar repetição exata no ensaio. Koito, Maloja e multi-scrobbler exigem verificação própria.

### Shutdown

Hoje `engine/engine.go:280–292` usa contexto único de 10 s, cancela ingestão antes de `httpServer.Shutdown` e chama `Fatal` em falha, pulando o `defer store.Close` da linha 145. Ordem proposta:

1. `httpServer.Shutdown` encerra admissão e espera handlers, com orçamento que reserva tempo para as etapas seguintes. Em erro, seguir a limpeza e retornar o erro.
2. Parar novos claims dos workers.
3. Aguardar o HTTP em voo por prazo menor que o orçamento restante, reservando margem para persistência.
4. Cancelar HTTP e, por CAS, liberar leases ainda possuídos como `retryable`, `next_attempt_at=now`, com `context.WithoutCancel` e deadline curto próprio. Preservar pausa do destino. Se o SQLite falhar, recuperar pelo lease expirado.
5. Aguardar os workers terminarem.
6. Fechar o store e retornar qualquer erro, substituindo `Fatal` por retorno.

Cumprir AC3 por persistência e recuperação, sem drenar a fila no shutdown.

## Migração 012 e D4

A 012 é necessária e aditiva. Acrescentar `ingest_events.delivery_policy` com DEFAULT legado e CHECK de domínio, imutável, estendendo as guardas 009 de sombra/modo. `outbox.format_version` distingue formatos, com valor legado para marcadores existentes. Última tentativa, status/classe e motivo podem usar `result`; estado e pausa do destino ficam em `integrations` ou tabela própria com FK composta. Nenhuma tabela de payload. Não fazer backfill ou reabrir `pre_082_best_effort`. Medir `ADD COLUMN ... CHECK` com 300 mil recibos. Usar EXPLAIN para claim/contagem/expiração antes de adicionar índices amplos.

Trocar cirurgicamente as condições relativas a relay em `trg_occurrence_cut_insert` e `trg_occurrence_cut_update` (011:24–41) e retirar as guardas `trg_occurrence_relay_insert` e `trg_occurrence_relay_update` (011:44–60) quando o caminho ativo estiver integralmente protegido pela política durável. Preservar drenagem v9, sombra limpa, origem, release, flag unidirecional e vedações a escritor legado/ocorrência autônoma. Em E2, deixar a substituição planejada e exercê-la em fixtures do store. Manter as guardas efetivas até E3 ligar o handler guiado pela política, para que nenhum executável intermediário permita relay bruto na via ativa de occurrence. Mestre desligado continua pausando novas admissões em perfil convertido.

Atualizar `currentSchemaVersion` (`bootstrap.go:22`), lista em `ingest_schema.go:21`, `validateIdentityObjects`, DDL embutido e sufixo de backup v11 em `backup.go`. Cobrir criação limpa, upgrade e validação em `validate_upgrade.py`, `validate_ingest_migration.py` e `validate_migration.py`; `LATEST_SCHEMA_VERSION` é derivado em `validate_multiuser.py:32`. Preservar WAL, FKs e `synchronous(FULL)`. Snapshot pré-upgrade inclui commits no WAL. Reversão só por backup v11 em diretório limpo, com o binário v0.47; preservar base nova para conciliar admissões posteriores e efeitos externos. Medir backup, DDL, boot, disco/WAL, RSS, lock e integridade. Testar interrupção do upgrade, reabertura e recusa de v12 pelo binário anterior.

## Verificação por AC

| AC | Verificação proposta |
| --- | --- |
| 1 | Replay desligado contra a tag v0.47, com single, import, playing_now e erros. Comparar URL, headers, bytes, ACK, escutas, sombra e exportação. Na via ativa validar lote misto, micro-lote de até 100, fixtures de payload, reenvio idempotente, v9/occurrence, recibo legado em voo no upgrade e marcador preservado, mudança de URL e de token, isolamento de dois perfis, fila no teto sem impacto no ACK e destino lento sem bloquear o cliente. |
| 2 | Stub com 429 e Retry-After nas duas formas, X-RateLimit-Reset-In e Remaining=0, valores inválidos/passados/overflow. Cobrir 5xx, timeout, conexão recusada, 401/403, 3xx e pausa; 400/413 por bisseção; 2xx com corpo grande ou erro de leitura; TLS inválido, IP privado, rebinding e validação pelo mesmo dialer. Relógio/jitter injetados, expiração, limite de 8, recuperação e justiça entre destinos. |
| 3 | SIGTERM com HTTP em voo e trabalho persistido; SIGKILL após admissão, após claim e após aceite remoto sem CAS; lease disputado com duas conexões; resposta atrasada após exclusão/revogação e SQLite cheio na conclusão. Confirmar ausência de perda local e medir repetição externa. |
| 4 | Registrar parecer original e emenda com comentário de autoria `advisor` no Backlog e registrar a revisão curta do contrato antes da E1. |

## Sequência e pontos de parada

- **E0.** Commit desta emenda e AC1. Parada para revisão curta do Advisor.
- **E1.** Transporte tipado e dialer SSRF em `internal/relay`, Sender legado intacto e testes de paridade, sem schema.
- **E2.** Migração 012, produtores v9 e occurrence, claim/CAS em lote, estado do destino, trigger de configuração, troca das guardas D4 em fixture e testes de integridade, duas conexões e upgrade. Parada Advisor para revisar invariantes.
- **E3.** Worker, lifecycle do engine, handler guiado pela política e corte efetivo das guardas D4 na 012 antes de publicar a migração. Parada Reviewer seguida de Advisor para encerramento e falhas ambíguas.
- **E4.** Ensaio C6/E4 com 300 mil recibos, destino saudável, lento e indisponível, rajada de 600 pedidos a 20/s, fila grande e vários perfis/destinos. Medir latência de ACK, tempo até decisão/entrega, throughput, contenção, retentativas por classe, CAS rejeitados, tempo de retomada e repetição no destinatário. Rodar `make check`, registrar evidência por AC, escrever CHANGELOG em inglês e passar ao Reviewer.

Riscos a verificar incluem aceitação parcial de um lote rejeitado, reentrega após crash, custos de bisseção e crescimento de falhas retidas, alteração de destino com HTTP em voo e impacto do egress em instalações privadas. UI da 083, agendador de fontes, cofre de credenciais, backfill e alterações de dedup ficam fora do escopo.

## Decisões do usuário em 20/11/2026

| ID | Decisão aprovada |
| --- | --- |
| D1 | AC1 com paridade byte a byte na via legada/sombra e em playing_now, entrega por evento na via ativa. |
| D2 | Política durável obrigatória para todo recibo ativo novo. |
| D3 | Derivação determinística do snapshot sem alterar Snapshot/Encode. A tabela concreta segue para validação final. |
| D4 | Micro-lote de até 100 itens por destino. |
| D5 | Backoff de 30 s, fator 2, full jitter, teto de 1 h, oito tentativas por item e 30 dias de idade, helper comum com a 173 e parâmetros por domínio. |
| D6 | 429 longo, 401/403 e 3xx pausam o destino. |
| D7 | Troca de URL leva pendências a falha reprocessável, troca só de token preserva linhas. |
| D8 | HTTP e relay privado só pela lista do root. Quem não é root vê somente classe HTTP. |
| D9 | Marcadores pré-082 fechados, com perda aceita de envios brutos interrompidos pelo upgrade. |
| D10 | 100 mil linhas abertas elegíveis por destino, excesso em falha recuperável, sem afetar ACK. |
| D11 | Falhas retidas indefinidamente até ação do operador. Mensagem e visibilidade em coordenação com a 083. |
