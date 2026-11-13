# Issue 037 — publicação automática e coleta de imagens

Concluída em 06/11/2026. [Issue](<../../tasks/issue-037 - 037-—-Coordenar-publicação-automática-e-coleta-de-imagens.md>) ·
[Revisão que identificou a corrida](ADVISOR_032_033.md).

## Correção

Os cinco caminhos que criam artistas ou álbuns com uma imagem automática passam
por `saveWithAutomaticImage`. Antes de baixar, a operação reserva o UUID no cache;
a reserva permanece até `SaveArtist` ou `SaveAlbum` retornar. O coletor verifica
as reservas sob a mesma trava de publicação e pula IDs em andamento. A trava só
é usada para registrar/remover a reserva e durante a própria coleta: não fica
retida durante o download ou o commit de associação.

As reservas têm contagem por UUID e liberação idempotente. O contexto pode
cancelar a aquisição e a espera do coletor. Se o download falhar sem cancelamento,
o caminho continua registrando o endereço da fonte, permitindo recuperação
posterior, como antes. `SkipCacheImage` mantém a associação sem baixar o arquivo.
Se o contexto for cancelado, a operação interrompe o download e não chama o save.

Um erro de save não apaga diretamente o arquivo: uma etapa posterior ao commit
pode ter falhado. Quando a reserva termina, o coletor volta a consultar o banco e
remove apenas arquivos sem associação. O fluxo administrativo existente continua
protegido por sua própria trava de publicação.

## Verificação

- A sequência antiga foi reproduzida com download local: coleta entre publicação
  e associação removeu o arquivo sem reserva. No fluxo corrigido, testes
  coordenados mantiveram duas imagens protegidas até a associação e verificaram
  que um órfão independente ainda é coletado.
- Duas requisições de download foram mantidas abertas ao mesmo tempo enquanto o
  coletor concluiu seu trabalho, demonstrando que a rede não segura a trava.
- O protocolo de reserva e coleta foi exercitado com commits reais de artista e
  álbum em SQLite em memória. Um teste adicional provocou erro em alias após o
  commit do artista e confirmou que a imagem associada permaneceu após a coleta.
- Testados cancelamento durante download e espera do coletor, falha de save com
  e sem associação, download indisponível, `SkipCacheImage`, contagem de reservas,
  liberação repetida e recuperação de source ausente pelo handler HTTP local.
- `CGO_ENABLED=0 go test ./...` passou; `go test -race` passou para catálogo,
  cache, engine e, separadamente, para os cenários finais de handler/coletor.
  Go 1.25.0 sem CGO passou nos testes 037; `go vet` passou nos pacotes alterados.
  Bancos eram em memória e servidores de teste usaram loopback.

## Limites

As reservas e a trava são mantidas **no processo**. Esta solução não coordena
múltiplos processos Koito que compartilhem o mesmo diretório de cache; esse
cenário exigiria mecanismo compartilhado próprio. Em um encerramento inesperado,
as reservas desaparecem; na próxima coleta, a decisão volta a ser feita pelas
associações persistidas no banco. O fluxo não limpa nem recodifica o cache e não
altera o esquema SQLite v3.
