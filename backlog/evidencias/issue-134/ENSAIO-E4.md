# ISSUE-134 — Ensaio da E4

## Método

Os binários foram compilados da tag v0.46 e da worktree rebaseada, com Go 1.26.3. Os ensaios rodaram em loopback em Mac16,1 arm64, macOS 15.7.3, 10 CPUs lógicas e 16 GiB de RAM. Cada cenário criou diretório e banco descartáveis. O modo por ocorrência foi exercitado apenas em perfis sintéticos sem relay. O teste do relay usou o caminho legado com os controles desligados.

Reprodução com os dois binários já compilados:

```sh
python3 scripts/rehearse_issue134_e4.py --binary /caminho/v11 --baseline /caminho/v046 --scenario load
python3 scripts/rehearse_issue134_e4.py --binary /caminho/v11 --baseline /caminho/v046 --scenario parity
python3 scripts/rehearse_issue134_e4.py --binary /caminho/v11 --baseline /caminho/v046 --scenario overlap
python3 scripts/rehearse_issue134_e4.py --binary /caminho/v11 --baseline /caminho/v046 --scenario fragmentation
python3 scripts/rehearse_issue134_e4.py --binary /caminho/v11 --baseline /caminho/v046 --scenario faults
go test ./internal/db/sqlite -run 'TestOccurrenceHistoryDeleteWithUnresolvedLease|TestOccurrenceAdmissionSQLiteFull' -v
```

## Volume no mesmo perfil e upgrade com WAL

A fixture v9 tinha 300.000 recibos ativos encerrados em accepted, com 300.000 escutas vinculadas, revisões e snapshots com título, artista, álbum, duração e instante. Distribuímos o histórico por 200 lançamentos. Preservamos as escritas no WAL antes do upgrade. Após converter esse mesmo perfil, enviamos 600 submissões com chaves distintas, a 20/s por 30 s, com MusicBrainz desabilitado. A observação de visibilidade consultou o SQLite a cada 25 ms. A amostragem de recursos inclui o backup no tamanho do diretório.

| Medida | Resultado |
| --- | ---: |
| Vazão observada até a última escuta visível | 19,39 escutas/s |
| Duração até a última escuta visível | 30,942 s |
| ACK p50 / p95 / p99 | 4,18 / 8,29 / 9,85 ms |
| Visibilidade desde envio p50 / p95 / p99 | 525,54 / 961,44 / 988,58 ms |
| Visibilidade após ACK p50 / p95 / p99 | 520,34 / 958,77 / 986,65 ms |
| Pico do processo, RSS amostrado | 50.384 KiB |
| Pico do diretório durante a carga | 814.609.224 bytes |
| WAL no início da carga e pico amostrado | 275.772.232 bytes em ambos |
| Upgrade até endpoint de saúde | 20,801 s |
| WAL v9 presente ao iniciar upgrade | 275.702.192 bytes |
| Tamanho do diretório quando o servidor ficou saudável | 813.575.822 bytes |
| Backup automático do v9 | 262.926.336 bytes |

As 600 admissões do fluxo receberam 200. Uma reserva de escritor mantida em conexão independente provocou 503 com `Retry-After: 1` em 5.066,2 ms. Após liberar a reserva, o reenvio da mesma chave gerou uma escuta. O tamanho físico do WAL ficou estável na janela de carga, com o arquivo já alocado pelo upgrade. O ensaio não estimou a quantidade de frames gerados ou reaproveitados.

O backup abriu com a tag v0.46, conservou os 300.000 recibos e passou em `foreign_key_check`. O mesmo binário recusou uma base v11 com `unsupported schema version 11`.

### Planos quentes medidos

- Ocorrência por release, release ausente ou track MBID usou `MULTI-INDEX OR`, com dois ramos em `idx_occurrences_release` e um no índice UNIQUE de track MBID. A ordem usa B-tree temporária.
- Release por texto usou `idx_release_aliases_alias` para o alias e chave primária para release. A distinção usa B-tree temporária.
- Seleção de claim/lease usou `idx_ingest_active_queue` por estado e B-tree temporária para ordenar. Os 300.000 recibos retidos estavam encerrados, fora da fila de admissão. Uma fila com 300.000 recibos elegíveis ou a expirar merece medição própria.

## Paridade, sobreposição e troca de modo

Com os controles desligados, v0.46 e v11 coincidiram nas projeções de 18 escutas e 18 recibos de sombra, origens, contagens e conteúdo exportado. O relay local recebeu a mesma rota, Authorization e corpo byte a byte com o perfil legado. O replay também passou após restaurar o backup.

Vinte escutas foram entregues primeiro pela API. Reenvios de cada uma chegaram por ZIP, Last.fm, Spotify, Maloja e TuneTrail/Koito. O resultado foi **20 escutas e 100 pending_duplicate**. Cada formato produziu 20 pendências `legacy_identity_collision`, todas com um único candidato tipado como evento. API, Last.fm e Maloja usaram segundos, ZIP e nativo milissegundos e Spotify microssegundos. A outbox ficou vazia. As frações distintas permanecem no recibo e a colisão na tripla ainda impede uma segunda escuta da ocorrência no mesmo segundo.

O caso A/B comprovado gerou duas escutas, uma gravação e dois lançamentos. A tentativa de voltar a flag para zero falhou. Após reiniciar com mestre desligado, o replay de B recebeu 503 com `Retry-After: 1` e manteve exatamente o recibo e a escuta anteriores.

## Fragmentação e exclusão em trânsito

- O stub MusicBrainz respondeu 503 para A/B da mesma gravação. Surgiram duas ocorrências e duas gravações provisórias. Com prova restabelecida, B foi promovida. A reapresentação comprovada de A ficou em `pending_identity` com `ambiguous_occurrence`. A 135 precisa reconciliar o vínculo provisório e a gravação comprovada conservando os dois lançamentos e o histórico já aceito.
- A resposta com atraso deliberado de 600 ms produziu uma escuta em 1,581 s desde o envio, incluindo a fila. A resposta retida produziu uma ocorrência provisória em 10,426 s, no primeiro intento. O cliente HTTP expirou dentro do orçamento de 20 s da prova e a ausência de evidência seguiu pelo catálogo textual.
- A compilação com mesmo título, artistas distintos e release MBID ausente produziu dois lançamentos. Dois títulos parecidos permaneceram distintos. Um alias exato compartilhado pelos dois lançou `pending_identity` com dois candidatos `release:`.
- A exclusão de histórico removeu a escuta vinculada. Uma entrega já reclamada, com occurrence_id ainda nulo, ficou em voo e criou um evento novo após concluir. O teste evita inferir seleção pelo título. Uma política diferente exige decisão sobre o escopo da exclusão de trabalho em trânsito.

## Falhas

A simulação de SQLite cheio limitou páginas na conexão de teste do escritor ativo. O lote de 70 itens devolveu `ErrUnavailable` e a transação deixou zero recibos e integração dessa entrega. Ao aumentar o limite, as mesmas chaves produziram os 70 recibos. O teste não ocupou o disco do sistema nem mediu falha de I/O do host.

## Limites e condições propostas para o piloto

A rajada local de 30 s usa um perfil, rede de loopback, catálogo novo repetido e um único hardware. Ela não mede dias de carga, disco remoto, serviços reais de MusicBrainz, fila acumulada de centenas de milhares de itens, concorrência prolongada com alterações de catálogo ou pico de disco durante o boot. Os valores são medidas observadas e precisam de calibração no volume de destino.

Para um piloto futuro, propor até 15 escutas/s sustentadas por perfil inicial, p95 de ACK até 200 ms e p95 de visibilidade até 2 s no fluxo sem espera de prova remota. Medir o fluxo com prova remota separadamente. Propor zero ACKs sem escuta ou recibo recuperável, zero contagens duplicadas em casos conhecidos, fila estável e taxa de 503 abaixo de 0,1% fora de manutenção. Reservar espaço de pelo menos quatro vezes o tamanho do banco de origem no volume de dados e verificar espaço temporário, incluindo backup, WAL e margem de segurança. Validar restore no disco de destino antes de habilitar um perfil. Suspender admissões e revisar a fila se os limites forem excedidos.

Pendências por tarefa:

- **135.** Leitores e mutações por ocorrência, exportação com posição, remoção da ponte/tripla, reconciliação provisória→comprovada e merge legado com reserva imediata.
- **083.** Consulta e revisão das pendências com evidência, autorização e CSRF para descarte, visibilidade para operadores e política para exclusão de trabalho em trânsito.
- **082.** Relay após decisão, entrega durável e idempotente. A condição sem relay continua valendo para este ensaio.
- **173.** Teto, intervalos, fila de falhas permanentes, alertas, reprocessamento e R-b.

Decisões do usuário ainda abertas: aprovar a janela, duração e limiares do piloto depois dos pré-requisitos, escolher o perfil inicial e o critério de suspensão, decidir se excluir histórico também deve cancelar admissões ainda sem ocorrência resolvida, e definir quem pode reconciliar gravações provisórias e reprocessar falhas terminais. As decisões D1–D9 continuam registradas no plano. Nenhum perfil real foi habilitado.
