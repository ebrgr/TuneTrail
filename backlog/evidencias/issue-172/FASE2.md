# ISSUE-172 — Evidências da fase 2

## Carga de ingestão

O ensaio usou uma cópia privada neutralizada, três chaves sintéticas no perfil
piloto e um mock ListenBrainz em loopback. O processo do TuneTrail ficou
restrito por sandbox à porta do mock para saídas de rede. A origem do ensaio
foi a cópia da fase 1. As 80 escutas de origem ficaram preservadas durante a
migração e a carga. O segundo perfil permaneceu com zero escutas durante a
carga principal.

| Medida | Limiar | Lote misto | Piloto completo |
| --- | --- | --- | --- |
| Taxa oferecida | 15/s | 15,00/s | 15,01/s |
| ACK p95 | < 200 ms | 11,43 ms | 11,60 ms |
| Visibilidade p95 | < 10 s | 0,969 s | 0,983 s |
| Visibilidade máxima observada | < 10 s | 1,067 s | 1,034 s |
| Respostas 503 | < 0,1% | 0/2.250 (0%) | 0/900 (0%) |
| Escutas aceitas ausentes do histórico | 0 | 0/2.045 | 0/900 |

O lote misto devolveu 2.045 respostas 200 e 205 respostas 400 por ausência
de artista. As 2.045 escutas aceitas ficaram visíveis. O total do perfil
piloto passou de 80 para 2.125. O piloto completo devolveu 900 respostas 200,
com todas as escutas materializadas. A visibilidade foi amostrada a cada
50 ms diretamente no histórico persistido.

Os casos adicionais terminaram com quatro escutas de arquivo aceitas, uma
pendência de duplicata com Revisar ligado e uma duplicata resolvida com
Revisar desligado. A falha forçada no resolvedor recebeu ACK 200 e ficou
estacionada após cinco tentativas (`retry_exhausted`).

O mock respondeu com um 429, com `Retry-After` de 12 segundos, e voltou a
aceitar entregas. A contagem final foi de 2.947 escutas no estado `sent`.
Quinze ainda estavam na fila quando terminou a medição do piloto completo.
Após reiniciar a instância isolada, as quinze foram entregues e a fila zerou.

## Comparação com o escritor legado

O binário anterior à troca da rota histórica recebeu as mesmas 2.250
requisições em outra cópia privada. Os dois escritores devolveram 2.045
respostas 200 e 205 respostas 400. As 2.045 projeções de perfil, segundo e
cliente tiveram o mesmo digest nos dois históricos, sem escutas exclusivas
de nenhum dos lados.

## Migração e restauração

- O teste de bootstrap v2 → v21 preservou 200 escutas, conferiu contagem por
  perfil e digest do histórico e materializou uma escuta nova. Revisar
  iniciou ligado.
- O teste v20 → v21 verificou que recibos de sombra sem escuta impedem a
  migração com um diagnóstico acionável. A nova tentativa preservou o
  histórico e o downgrade foi bloqueado.
- Em uma segunda cópia privada de desenvolvimento v20, o backup SQLite
  passou por `quick_check`. A migração para v21 manteve 2.826 escutas.
  A restauração foi feita em um diretório novo e aberta pelo binário v20.
  As 2.826 escutas, as contagens e os digests por perfil permaneceram iguais.
  Também foi conferida a projeção dos campos de histórico, e
  `quick_check` retornou `ok` após a restauração.

## Verificação da branch

O `make check` passou, com 13 de 13 jornadas E2E. A suíte completa
`go test -race -count=1 -timeout 30m ./...` terminou com sucesso.

## Riscos operacionais

A reversão exige o backup anterior ao corte. Escutas recebidas depois do
backup ficam no banco atualizado. A janela de 10 minutos deve reservar
tempo para verificar a cópia antes de retomar os clientes.

Um destino novo ou alterado fica em rascunho até a aprovação de rede.
Escutas recebidas antes da aprovação ficam no histórico local. Para
destinos nomeados, a confirmação valida as respostas DNS contra a política
existente. Um destino nomeado em rede restrita pode precisar de uma
permissão de administrador para o host, a porta e o CIDR antes de ativar.

A medição cobre a máquina de ensaio, uma cópia de desenvolvimento e um
relay em loopback. O teste de salto v2 usa uma fixture. Os tempos e o limite
de ocupação da fila em produção dependem da carga e do destino configurado.
