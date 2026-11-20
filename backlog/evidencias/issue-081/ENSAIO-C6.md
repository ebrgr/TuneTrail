# ISSUE-081b — Ensaio sintético C6

## Resultado para aceite

O fluxo regular conservou as 410 escutas esperadas nos quatro cenários e as três
escutas adicionais da troca de controles. O reensaio de R1 recuperou a escuta
ausente e preservou as já materializadas. A falha de COMMIT devolveu 503 e
preservou os recibos anteriores. Os cenários verificados sustentam o envio
para aceite. A promoção do piloto depende dos limiares e da janela aprovados
pelo usuário.

## Método

O [ensaio reproduzível](../../../scripts/rehearse_issue081.py) usa servidores
TuneTrail de verdade em loopback, bancos temporários e dados fictícios. A base
v0.43 veio da tag `v0.43` (`4f9232e`). O binário v9 foi compilado a partir do
código aprovado até `6b5fd6c` para a carga e as outras medições. O reensaio de R1 usou o binário com `50b6e83`. A máquina foi um Mac16,1 arm64 com 16 GiB, Go
1.26.3 e Python 3.15.2. O gerador da ISSUE-133 prepara 30 mil escutas, quatro
perfis, 9.018 faixas e recibos de sombra. O ensaio copia a mesma base para cada
cenário. A confirmação de ativação nos cenários indicados usa a API root real.

Cada cenário enviou 160 escutas ListenBrainz, em oito clientes concorrentes, e
cinco arquivos independentes de 50 escutas cada. Os arquivos cobrem Spotify,
Last.fm, Maloja, Koito e ListenBrainz ZIP. O histórico foi consultado a cada
20 ms. O ensaio compara perfil, segundo, cliente, título e origem observados
com a projeção da v0.43. O observador também captura o tempo entre o envio, a
resposta e a primeira observação no histórico. Os picos amostrados a cada
25 ms usam RSS do processo e soma dos tamanhos lógicos no diretório de dados,
incluindo WAL e backup. O WAL é exibido como subconjunto. Arquivos temporários
fora do diretório, arquivos já desvinculados e logs ficam fora da soma.

Para reproduzir, compile os binários da tag e desta worktree, então execute:

```sh
python3 scripts/rehearse_issue081.py --binary /tmp/tunetrail-v9 --baseline /tmp/tunetrail-v043 --scenario workload
python3 scripts/rehearse_issue081.py --binary /tmp/tunetrail-v9 --scenario purge
python3 scripts/rehearse_issue081.py --binary /tmp/tunetrail-v9 --scenario replay
python3 scripts/rehearse_issue081.py --binary /tmp/tunetrail-v9 --scenario faults
```

## Carga e troca de modo

| Cenário | ACK API por s | ACK API p50/p95/p99 (ms) | Escutas API visíveis por s | ACK → visível p95 API (ms) | Importadas admitidas por s | Importadas visíveis por s | ACK → visível p95 importação (ms) |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| v0.43, schema 8 | 75,6 | 104,34 / 109,34 / 119,77 | 75,3 | 22,52 | 74,3 | 74,5 | 0 |
| v9, mestre desligado e perfil ligado | 75,3 | 104,99 / 111,38 / 119,18 | 74,7 | 21,78 | 74,0 | 73,8 | 0 |
| v9, mestre ligado e perfil desligado | 75,0 | 105,36 / 111,09 / 120,80 | 74,8 | 22,66 | 74,2 | 74,2 | 0 |
| v9, ambos ligados | 1.849,7 | 0,60 / 18,66 / 54,14 | 31,1 | 4.988,57 | 215,6 | 31,7 | 6.585,60 |

A v9 ativa respondeu aos arquivos em 8,5–12,1 ms para Spotify, Last.fm, Maloja
e ListenBrainz ZIP. Koito levou 1,12 s e inclui associação ao catálogo na
própria importação. Na v0.43, os cinco arquivos levaram 0,67–0,68 s cada.
Após o ACK ativo, API e arquivos chegaram ao histórico com p99 de 5,10 s e
6,71 s. O volume de 1.849,7 ACK/s foi uma rajada local. A vazão observada de
materialização ficou perto de 31 escutas/s.

| Cenário | Dados e backup (MiB) | WAL (MiB) | RSS (MiB) |
| --- | ---: | ---: | ---: |
| v0.43 | 27,9 | 4,0 | 44,5 |
| v9, mestre desligado | 54,7 | 9,0 | 55,5 |
| v9, perfil desligado | 50,0 | 4,0 | 54,4 |
| v9 ativo | 54,7 | 9,0 | 57,4 |

As 410 escutas tiveram a mesma projeção e o mesmo total nos quatro cenários.
No modo ativo, 410 recibos ficaram em `accepted`. Em seguida, desligar o
perfil, religá-lo via prévia e reiniciar com o mestre desligado preservou as
três novas escutas. Com o modo ativo de novo, reenviar a mesma faixa no mesmo
segundo sem chave resultou em `duplicate` com `review_deferred`, sem aumentar
a contagem.

## R-a — Falha determinística e política de retry

Um trigger sintético abortou a gravação da revisão, depois da tentativa de
criar a escuta na mesma transação. A API já tinha respondido 200. O recibo
voltou para `retryable` em 0,99 s, 60,99 s e 121,00 s. O contador avançou de
1 a 3 e a base permaneceu sem escuta ou revisão para aquele item. Removido o
trigger, a quarta tentativa aceitou a escuta em 180,98 s. O intervalo medido
foi de um minuto, e o código atual não usa o contador como teto.

Proposta para aprovação: até cinco tentativas automáticas para falhas
transitórias, com espera de 1, 2, 4 e 8 minutos entre elas. Na quinta falha,
reter recibo, chave, motivo e contagem numa fila durável de falhas permanentes,
com alerta e reprocessamento explícito após correção. Encerrar erros de
entrada reconhecidamente determinísticos com motivo registrado; preservar a
possibilidade de recuperação de todo recibo confirmado.

## R5 — MusicBrainz lento e retomada por lease

O servidor MusicBrainz sintético manteve a resposta aberta. O cliente real
aplica timeout de dez segundos por requisição. A admissão respondeu em
0,74 ms e a escuta foi materializada 20,04 s depois da primeira chamada
MusicBrainz, ainda na primeira tentativa e dentro do lease de 120 s.

Uma segunda escuta recebeu 200. Depois de o worker obter o lease e entrar
na chamada externa, o ensaio matou o processo com SIGKILL. Nesse ponto
havia um recibo durável, uma tentativa e nenhum vínculo de evento. O ensaio
reiniciou com o mestre desligado e também desativou o perfil antes do
vencimento. A escuta foi materializada 120,16 s após a morte do processo,
na segunda tentativa. As duas escutas confirmadas tiveram um único evento
cada. O teste Go com duas conexões comprova que o CAS rejeita o lease
anterior após a retomada. O stub não representa uma cadeia externa cujo
tempo total ultrapasse 120 s com o worker original ainda ativo.

## R7 — Purga e BUSY

O perfil selecionado continha 300 mil recibos de sombra e 30 mil observações.
A prévia levou 2,061 s. A confirmação, que recalcula o digest sob
`BEGIN IMMEDIATE`, exclui os recibos e suas observações e faz commit, levou
4,079 s de ponta a ponta. Outro escritor com timeout de 25 ms registrou 82
tentativas BUSY. Da primeira tentativa bloqueada ao fim da última passaram
3,961 s. As 30 mil escutas preexistentes foram preservadas. Os picos foram
220,9 MiB em dados, 74,2 MiB em WAL e 58,5 MiB de RSS.

Separadamente, um escritor sintético manteve o bloqueio além do timeout
normal da aplicação. Quatro admissões concorrentes receberam 503 em
5,064–5,065 s com `Retry-After: 1`. Reenviadas após o desbloqueio, as quatro
receberam 200 e produziram quatro escutas.

## R1 e R-b — Replays na fronteira da ativação

- **Reinício do mestre.** Com o perfil opt-in e o mestre desligado, uma
  entrega com chave gravou sombra e escuta. Um trigger sintético fez outra
  entrega gravar sombra e falhar na escrita legada com HTTP 500. No ensaio
  original, o replay ativo dessa segunda chave recebeu 200 e ficou sem escuta.
  No reensaio, uma falha de COMMIT durante a substituição devolveu 503 e
  preservou os dois recibos de sombra. Removida a falha, os dois reenvios
  receberam 200. O resolvedor marcou a primeira chave como `duplicate` pelo
  evento legado e aceitou a segunda, recuperando a escuta ausente. Restaram
  zero recibos de sombra sem vínculo e zero confirmações sem escuta após
  a resolução. O recibo ativo de controle também foi resolvido.
- **Job iniciado em sombra.** Uma importação Spotify de 160 escutas começou
  antes da ativação. A página final gravou 160 recibos de sombra depois da
  purga. O upload ativo seguinte substituiu as 160 chaves e o resolvedor
  marcou todas como `duplicate`. As 160 escutas já existentes continuaram
  no histórico, cada uma uma vez.
- **Cobertura de admissão.** Os testes Go verificam os três índices de chave
  compartilhados, um lote com recibos substituídos, preservados e novos,
  cascata de observações e detalhes, rollback do lote e isolamento entre
  perfis. A substituição preserva a hora estimada original quando um
  reenvio com a mesma chave continua sem timestamp.
- **R-b.** Ocultar temporariamente a tabela de flags com o mestre ligado e
  o perfil inativo provocou HTTP 500 em 6 ms, sem escuta. Restaurada a tabela,
  o reenvio escreveu pelo caminho legado. O teste isola um erro genérico
  anterior à leitura do modo.

## Recomendações e decisões de promoção

Para a promoção, propor três semanas de medição em sombra, cobrindo fins de semana,
picos e importações. Aplicar o piloto a um perfil com até 15 escutas/s
sustentadas, aproximadamente metade da vazão local observada. Propor zero
confirmações sem escuta ou trabalho recuperável, zero contagens duplicadas
indevidas, p95 de ACK abaixo de 200 ms, p95 de visibilidade abaixo de 10 s e
menos de 0,1% de respostas 503 fora da manutenção. São limites iniciais a
aprovar e calibrar na instalação de destino.

Reservar uma janela inicial de dez minutos para parar escritores, concluir jobs,
verificar backup, exportar a prévia, confirmar e testar um envio. O bloqueio
medido foi de cerca de quatro segundos. Repetir a medição com o volume e o disco
do destino antes de marcar a janela de produção.

Decisões do usuário que permanecem abertas

- Aprovar os limiares, a duração da sombra e a janela de manutenção.
- Definir a resposta a R-b quando o modo efetivo ainda é indeterminado.
- Aprovar o limite de tentativas e a operação da fila de falhas permanentes.
- Coordenar a remoção do escritor legado com a ISSUE-172, que já existe na
  `main`. Decidir se a fila de revisão e falhas permanentes cabe na ISSUE-083
  ou exige tarefa própria. A opção `review=true` continua bloqueada.

Os dados sintéticos não medem a incidência de R1, carga sustentada em produção,
todas as latências de catálogo remoto ou a distribuição de falhas de disco.


## Verificação e critérios

`make check` passou por completo, incluindo 13/13 jornadas e2e. As jornadas
que verificam o schema vigente passaram a derivar a versão das migrações
do checkout. A validação adicional incluiu a sintaxe do script Python e
da navegação dos guias.

- AC1 foi verificado pela comparação das 410 escutas, pela troca dos
  controles e pelo reensaio de R1. Os reenvios recuperaram a escuta
  ausente e mantiveram a contagem das já salvas.
- AC3 foi verificado com repetições legítimas, testes de concorrência,
  troca de flag com um job fixado, retomada por lease, fila parcial com
  EXPLAIN e exclusão seletiva de recibos. O reensaio de R1 confirmou
  a transição sem contagens extras nos dois cenários.
- AC2 foi verificado pelo teste de colisão sob reserva de escritor, pelo
  motivo `review_deferred` preservado no ensaio e pelo teste de decisão com
  intenção de relay. A falha injetada no ensaio de retry comprova rollback
  do evento antes de uma nova decisão.
- AC4 foi verificado pelo bloqueio de `review=true` e pelo fallback testado.
  O guia de ativação registra que a fila do operador ainda precisa ser
  entregue. A dependência funcional é a ISSUE-083.
- AC5 foi verificado no histórico da tarefa. O parecer high e suas
  ressalvas foram registrados e o Reviewer aprovou a releitura da
  correção antes da C6.
