# ISSUE-082 — Ensaio E4 do relay durável

## Método e reprodução

Apple M4 arm64, 16 GiB, macOS 15.7.3, SSD local, Go 1.26.3 e Python 3.15.2. Um perfil sintético na carga, vários perfis fictícios no upgrade. Destinos HTTP/TLS em loopback, com DNS injetado em testes e sem serviços externos. Diretórios temporários novos e independentes. A carga saudável e os ensaios menores rodaram parcialmente em paralelo com validações locais, sem controle térmico ou repetição estatística. Os percentis de ACK usam 40 submissões unitárias por cenário. Os percentis de import têm apenas duas ou quatro observações.

```sh
e4_dir=$(mktemp -d)
mkdir "$e4_dir/src-v048"
git archive v0.48 | tar -x -C "$e4_dir/src-v048"
(cd "$e4_dir/src-v048" && go build -o "$e4_dir/v048" ./cmd/api)
go build -o "$e4_dir/current" ./cmd/api
python3 scripts/rehearse_issue082_e4.py --binary "$e4_dir/current"
python3 scripts/rehearse_issue082_e4.py --binary "$e4_dir/current" --cases capacity
python3 scripts/rehearse_issue082_e4.py --binary "$e4_dir/current" --baseline "$e4_dir/v048" --cases upgrade
python3 scripts/validate_relay_parity.py --baseline "$e4_dir/v048" --binary "$e4_dir/current"
python3 scripts/validate_relay_crash.py --binary "$e4_dir/current"
python3 scripts/validate_shutdown.py --binary "$e4_dir/current"
python3 scripts/rehearse_issue082_tls.py
TUNETRAIL_RELAY_MEASURE=1 go test ./internal/db/sqlite -count=1 -v \
  -run 'TestRelay(E4Store|MutationMeasurements|MigrationMeasurements|MarkerMeasurements|Capacity|Upgrade|BootstrapWAL|LegacyDNS|FullCompletion)'
go test -race ./internal/relay ./internal/relayoutbox -count=1
make check
```

## Carga, ACK e entrega

| Destino | Janela de observação | POSTs no destino | Resultado | ACK unitário p50 / p95 / p99 |
| --- | ---: | ---: | --- | --- |
| Saudável, com limite preventivo e um item inválido | 127,210 s até resolver e entregar | 140 | 4.039 enviados, 1 rejeição isolada, 4.040 escutas locais | 0,51 / 0,98 / 2,46 ms |
| Lento, cabeçalhos após 7,7 s | 24,410 s até entrega | 3 | 160 enviados, maior lote 100 | 0,49 / 0,80 / 1,42 ms |
| Porta recusada | 6,215 s até confirmar pausa e conservação | 0 | 128 pending e 32 retryable, sem perda | 0,49 / 0,71 / 1,85 ms |
| Oscilante, HTTP 503 → 429 com Retry-After de 2 s → 200 | 71,476 s até entrega | 4 | Pausas server e rate_limited, depois 160 enviados | 0,52 / 0,67 / 1,60 ms |
| Porta recusada, cheio após preenchimento | Sem drenagem durante a pausa | 0 | 100.000 abertos e 40 `failed/queue_capacity`, 100.040 escutas locais | 0,53 / 0,75 / 1,60 ms |

A origem de carga saudável simulou quatro arquivos de 1.000 escutas, submetidos simultaneamente como quatro corpos ListenBrainz `import`. O ACK de import teve p50 de 363,04 ms e p95/p99 de 816,25 ms. O uploader administrativo de histórico não cria entregas de relay e ficou fora da carga. Dos 140 POSTs, 124 continham 32 itens. A sequência de lotes menores é compatível com a bisseção que separou o único 400 determinístico. A pausa preventiva foi observada após 2xx com Remaining=0/Reset-In=2. O mock não recebeu duplicatas em nenhum cenário de carga. Houve 2.020 inversões adjacentes na ordem de carimbos no cenário saudável, portanto o piloto não deve depender da ordem.

A saturação manteve 200 para todos os 40 ACKs depois do teto. No mesmo fixture, uma reserva externa do escritor causou 503 em 5,061 s. Essa última medida isolou SQLite BUSY e não simulou esgotamento físico do disco na admissão. O teste de SQLite FULL cobre a gravação do 2xx no store e confirmou que o claim e o contador sobreviveram. No ramo de NXDOMAIN, a resposta DNS injetada no transporte gerou diagnóstico transitório. A pausa de 30 s na semeadura foi verificada separadamente com resolvedor de teste.

| Cenário | RSS pico | Arquivos de dados pico, incluindo WAL | WAL pico | CPU em 3 s após observação |
| --- | ---: | ---: | ---: | ---: |
| Saudável | 69.392 KiB | 12.923.272 B | 4.280.712 B | 0,14 s |
| Lento | 48.384 KiB | 5.103.792 B | 4.243.632 B | 0,21 s |
| Recusado | 49.840 KiB | 5.087.288 B | 4.223.032 B | 0,07 s |
| Oscilante | 48.592 KiB | 5.083.240 B | 4.231.272 B | 0,08 s |
| Cheio | 52.608 KiB | 171.085.472 B | 56.888.992 B | 0,24 s |

A CPU é a do processo inteiro na janela posterior à carga, com quatro workers e polling de 250 ms. Pode incluir atividades residuais e coleta de lixo. Este ensaio não atribui o custo por worker.

## Banco grande e manutenção

Com 300.000 recibos no perfil e 100.000 itens abertos, sete ticks de `ExpireRelayWork` sem vencidos levaram 15,55–16,29 ms. O EXPLAIN do subselect aberto usou `idx_outbox_relay_due` por perfil, destino e geração, filtrando a idade. O subselect de capacidade usou `idx_outbox_relay_capacity` incluindo `created_at<?`. O UPDATE externo procurou os IDs em `sqlite_autoindex_outbox_1`. O resultado ficou abaixo do limiar de 50 ms, sem alteração da 012.

- A outbox ocupou 34.287.616 B ou cerca de **343 B por item aberto**, incluindo a tabela e todos os índices. A estimativa foi confirmada pela diferença de páginas em uso ao apagar a fila dentro de um savepoint revertido. A tabela sozinha usou 10.227.712 B. Os índices due, capacity e legado queue usaram 4.464.640, 4.096 e 2.940.928 B. O restante está nos índices de evento e unicidade. A amostra não estima o custo de linhas sent/failed no longo prazo, nem inclui o snapshot no recibo.
- Readmissão de um overflow na próxima reserva, com claim incluído, levou 0,783 ms. O contador voltou ao teto e bateu com a contagem real. O ensaio não extrapola esse número para milhares de overflow.
- Exclusão de 100.000 escutas de uma faixa, com preview e aplicação, levou 6,080 s no fixture combinado e zerou o contador. Deve ser tratada como manutenção que pode disputar o escritor. Com 100.000 abertos e o fixture menor, excluir o perfil levou 2,227 s, trocar URL 0,261 ms e remover o relay 0,121 ms. Nas duas últimas operações a geração mudou sem varredura imediata dos itens.

## TLS e entrega pelo menos uma vez

TLS real em loopback, certificado temporário verificado com CA local só no cliente Python. O protótipo compara conexões fechadas após cada POST com uma conexão por claim. Cada linha mede 3.000 claims, com 100 itens no total por claim. Não mede WAN, certificado público, SSRF do protótipo nem o tempo total da pipeline Go.

| POSTs por claim | Sem reutilização | Reutilização só no claim |
| ---: | --- | --- |
| 1 | 3.000 handshakes em 3,485 s | 3.000 em 3,145 s |
| 4 | 12.000 handshakes em 12,469 s | 3.000 em 4,181 s |

Manter DisableKeepAlives para o piloto. A reutilização só dentro do claim não elimina handshakes no caso comum de um POST. A vantagem aparece com bisseção repetida. Medir com RTT realista e frequência operacional de bisseção antes de decidir.

| Fronteira | Recebimentos aceitos no mock | Chaves únicas | Repetições |
| --- | ---: | ---: | ---: |
| SIGKILL depois do ACK persistido, antes da resolução | 1 | 1 | 0 |
| SIGKILL depois do claim, antes do aceite remoto | 1 | 1 | 0 |
| SIGKILL após aceite remoto, sem CAS | 2 | 1 | 1 |
| Falha injetada na gravação do 2xx e retomada | 2 | 1 | 1 |
| SIGTERM em voo, mock conta o recebimento antes de responder | 2 | 1 | 1 |

O script de SIGKILL adiantou offline o vencimento persistido do lease após a queda, para evitar uma espera de dois minutos. O mesmo evento persistiu localmente uma vez. No SIGTERM fora do Compose, a saída levou **4,011 s**, o lease foi liberado sem consumir tentativa e a segunda execução concluiu a entrega. O mock consolidou as repetições por (listened_at, faixa, usuário). Isso valida a chave simulada, não a implementação de ListenBrainz, Koito ou de outras instâncias. Maloja e multi-scrobbler continuam **a confirmar**.

## Upgrade, paridade e segurança

Com o fixture da 134 de 300.000 recibos e WAL aberto de 278.219.512 B, a v12 ficou pronta em **21,052 s**. A base final ficou em 276.852.736 B. O backup físico `.pre-relay-delivery-*.bak` ocupou 265.453.568 B. A restauração em diretório novo abriu na v0.48 com schema v11 e os mesmos 300.000 recibos. A v0.48 recusou v12 no replay de paridade. O ensaio isolado com 300.000 marcadores obteve backup de 240.709.632 B em 0,790 s e migração em 2,343 s com WAL de 86.552 B. O boot grande inclui manutenção e subida do HTTP e usa outra distribuição de dados.

O upgrade real em loopback semeou as duas exceções do administrador e deixou o perfil comum pausado como `policy_blocked`, com `seed_skipped_non_admin` no log. O teste com DNS injetado verificou falha transitória com retomada elegível após 30 s, sem semeadura, separadamente do upgrade grande. O teste de paridade via v0.48 comparou caminho com prefixo, cabeçalhos e corpo byte a byte para single, import e playing_now, duas respostas de erro e ausência de POST nos erros. A via desativada inclui o registro sombra quando admitido.

Os testes do dialer cobriram loopback, privado, link-local, metadados, rebinding, redirects, global ligada/desligada e exceções. A saída para não-root mantém uma classe única após DNS, salvo autenticação em IP público. Logs sanitizados omitem IP resolvido, caminho, query e token. A fila, as tabelas de entrega, o GET administrativo e a exportação de histórico não expõem a credencial. **O backup físico de rollback contém a credencial**, confirmado por consulta na fixture, conforme o contrato anterior de restauração integral. A exigência de ausência de token em qualquer backup não foi atendida. O piloto precisa de uma decisão explícita sobre backup físico integral, backup expurgado ou armazenamento protegido de credenciais. Nenhum valor da fixture foi registrado nesta evidência.

## Critérios e piloto

- **AC1 reverificado.** Eventos aceitos da API viraram entregas, o import ativo terminou sem perda e a fila cheia manteve ACK 200. Paridade observada contra v0.48 e golden legado v0.47 já coberto pelo teste do transporte.
- **AC2 reverificado.** 503 e 429 repetiram com pausa e recuperação. Refused e NXDOMAIN foram transitórios. Os testes de transporte verificaram Retry-After em segundos/data, Reset-In e redirecionamento bloqueado.
- **AC3 reverificado.** SIGTERM e SIGKILL preservaram dados. Houve as reentregas esperadas nas fronteiras ambíguas.
- **AC4 segue pendente do aceite.** A pendência de segurança do backup físico impede mover a tarefa para Aguardando aceite nesta rodada.

Para um piloto no mesmo tipo de host, começar com até **15 eventos/s por perfil** de modo sustentado. A carga saudável sustentou 31,75 eventos/s da admissão até a conclusão, incluindo a resolução v9. Investigar **ACK unitário p95 acima de 50 ms** durante cinco minutos sob carga comparável, separando ACK de import. Usar **150 s para drenar 4.000 eventos** em destino saudável como referência inicial. Alertar com 50.000 itens abertos ou 20 MiB de páginas de outbox por destino, e escalar a 80.000 ou 30 MiB. Monitorar capacidade e disco também fora dessas referências. O teto de 100.000 abertos não é um orçamento de disco, pois há recibos e histórico fechado. Reservar espaço para base, WAL e backup coexistirem no upgrade, com margem extra em disco mais lento.

Para a 083 ficam retenção de sent/failed, consultas agregadas de estado e UI para erros/readmissão. Decisões do usuário incluem a política de credenciais em backups físicos e a janela operacional para exclusão grande. Ficam por medir sob carga operacional latência de WAN, contenção prolongada entre muitos perfis, crescimento de filas fechadas, NAS/SD, falha física de disco no ACK e deduplicação de receptores reais. Nenhum código de produção foi alterado.
