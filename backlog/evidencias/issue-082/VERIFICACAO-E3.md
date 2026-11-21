# ISSUE-082 — Verificação da E3

Ensaios em macOS arm64, com SQLite descartável, HTTP em loopback e credenciais sintéticas. A comparação legada usou um binário compilado da tag `v0.48` e um compilado da worktree. A E4 ainda fará a medição de carga e interoperabilidade com destinos reais.

| AC | Evidência |
| --- | --- |
| 1 | `validate_relay_parity.py` comparou método, URL, cabeçalhos, corpo byte a byte e ACK para single, import e playing_now, além de dois erros. O binário v0.48 recusou abrir o banco v12. `TestDurableRelayAdmission` cobriu lote misto, replay idempotente e recibo legado em voo que manteve o marcador. `TestRelayOccurrenceIntent` e `TestRelayOccurrenceConversion` cobriram decisões pendentes, escritor occurrence e corte com destino ausente, bloqueado e permitido. `TestRelayCapacity` admitiu com 100 mil itens e verificou overflow em falha readmitível. `TestSlowRelayDoesNotBlockOtherProfile` completou a segunda entrega enquanto a primeira ainda aguardava HTTP. |
| 2 | Testes de transporte cobrem 429, Retry-After em segundos/data, X-RateLimit-Reset-In, limite e overflow, resposta 2xx grande/truncada, erro TLS, recusa de conexão, timeout, redirects, endereço reservado e rebinding. `TestRelayTransientPauses` verificou relógio e jitter injetados e pausas com ou sem prazo. `TestWorkerBisectsAndSettles` isolou um 400 e confirmou os demais itens. Testes do diálogo administrativo exigiram as duas permissões para HTTP privado e verificaram o mascaramento do token. |
| 3 | `validate_shutdown.py` enviou SIGTERM com relay em voo e retomou a entrega após reiniciar. No banco após shutdown havia trabalho retryable sem lease, tentativa gasta ou pausa. `validate_relay_crash.py` matou após admissão, após claim e após aceite remoto antes da confirmação local. Todas as escutas permaneceram. O terceiro cenário observou duas aceitações no stub. O ensaio antecipou o vencimento dos leases no SQLite com o processo parado para evitar esperar pelo prazo real. Os testes do store cobriram disputa de lease, resposta tardia após troca de URL e exclusão, SQLite cheio no CAS, e remoção de histórico por faixa, álbum e artista. |
| 4 | A revisão de encerramento e falhas ambíguas pelo Advisor permanece pendente após o Reviewer. |

O arquivo de sent após exclusão contém somente perfil, dia UTC e contagem. Os testes confirmam a eliminação do arquivo em `PurgeAllData`, a retenção diária de 30 dias e a ausência de horário por escuta na tabela. O ensaio de crash também verificou que o token do destino não aparece na exportação lógica.

A configuração durável e a conversão occurrence são cercadas por `config_version`. Um 2xx que retorna da URL antiga depois de a configuração mudar pode ter produzido efeito remoto e deixa o item antigo em `failed/destination_changed`. Retry explícito envia ao destino vigente. Exclusão antes da verificação de envio retira a linha do lote. Uma exclusão não desfaz um pedido já em voo.

```sh
make check
python3 scripts/validate_relay_parity.py --baseline /path/to/v0.48 --binary ./tunetrail
python3 scripts/validate_relay_crash.py --binary ./tunetrail
```
