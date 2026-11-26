# Parecer da ISSUE-214: subsistema de enriquecimento

Revisão de 26/11/2026 do diff `main...HEAD` da branch `chore/issue-214-subsistema-de-enriquecimento-interface-unica`, sobre a v0.76 (schema 24, migração nova 025).

**Esta revisão é do próprio implementador, executando o papel do Reviewer. Não é independente.** O Advisor (effort high), que a issue pede por tocar fila, concorrência e chamadas externas, é acionado pelo usuário e não foi executado.

## Achados

1. `[RISCO]` `internal/db/sqlite/enrichment.go`, `bindingFor`: a elegibilidade `profile-relay` seleciona o vínculo do ListenBrainz. Uma segunda fonte com relay reaproveitaria o vínculo errado. Corrigido com comentário que documenta o acoplamento; a segunda fonte precisa do próprio vínculo.
2. `[BUG, corrigido]` custo de bytes. A primeira versão gravava uma linha por recibo e provedor. Medido com 5.000 escutas: 260 B por escuta (11% dos 2.382 B), quase tudo «nada a acrescentar». Agora só se grava encontrada, ambígua e falha (commit `ee53cca`).
3. `[RISCO]` `enrichment_listenbrainz.go`, `prepareListenBrainz`: sem o resultado em memória do Navidrome, a etapa do ListenBrainz lê as declarações já gravadas. Num caso de borda (o Navidrome já declarou um artista em escuta anterior e só falta a gravação) a consulta ganha o parâmetro `metadata=true&inc=artist`, que antes não ia. Muda só o formato do pedido, não o resultado; não há teste que fixe isso.
4. `[RISCO]` um job recuperado pergunta de novo às fontes cuja resposta «nada a acrescentar» não foi gravada. Limitado a duas tentativas por recibo e aceito em troca dos 260 B por escuta.
5. `[RISCO]` uma resposta «ausente» de um provedor não persiste, então não existe cache negativo entre escutas. Hoje o cache em memória de 10 minutos de cada provedor cobre isso; a persistência faz sentido com o MusicBrainz por nome (ISSUE-216).

## Verificado

- Testes das ISSUE-174 e 175 passam sem alteração de comportamento (só mudou a assinatura interna de `prepareListenBrainz`, que perdeu o parâmetro `primary`).
- Testes novos: 10 de cadeia e registro no banco, 7 no pacote `enrichment`, 2 de orçamento do MusicBrainz, 2 do registro no motor e 1 de migração (24 para 25, com e sem Navidrome configurado). Verificação por mutação: sem a regra de resposta final, os testes de recuperação e de falha transitória falham; sem a regra de fonte própria, o teste de fonte falha.
- `go test -race` limpo nos pacotes `internal/db/sqlite`, `internal/enrichment` e `internal/mbz`.
- A trava de bancos de preview continua recusando 24 (pela tabela de preview), 25 (sem o registro) e 26 a 28.
- `make check` com 13 de 13 percursos de ponta a ponta.

## Não verificado

- Navidrome e ListenBrainz reais ponta a ponta: os testes usam servidores simulados.
- Banco grande: só medi o custo em bytes numa tabela sintética de 5.000 escutas.
- Comportamento sob carga de importação (dezenas de milhares de recibos na fila).

## Decisão pendente do usuário

Acionar o Advisor (effort high) antes do aceite, como pede a issue? Os pontos que eu mais gostaria de ver auditados são a renovação do lease entre etapas, a ordem de gravação em relação ao fence e o fato de o ListenBrainz passar a ler evidência persistida.
