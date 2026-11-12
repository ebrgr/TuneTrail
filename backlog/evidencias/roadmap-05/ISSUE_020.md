# Issue 020 — concorrência, falhas e I/O

Execução em 05/11/2026, Go 1.26.3, darwin/arm64, SQLite v3. Os testes usam
bases descartáveis, credenciais fictícias e servidores HTTP locais. As verificações
abaixo descrevem cenários específicos, não uma prova exaustiva das intercalações.

## Correções e regressões

- A ingestão ListenBrainz aceita até 16 MiB por corpo e até 1000 escutas por
  ocorrência de `payload`. Corpos maiores retornam 413, inclusive sem
  Content-Length; o parser interrompe o processamento no item 1001. O limite é
  aplicado após autenticação. Campos desconhecidos e envelopes anteriormente
  suportados foram exercitados. A chave de coalescência agora utiliza a estrutura
  completa, evitando colisões entre fronteiras de campos.
- Falha ao associar artistas ao álbum interrompe o salvamento da escuta. Metadados
  eventualmente criados antes da falha não são uma transação integral.
- A fila cancela trabalho cooperativo, resolve solicitações pendentes ao encerrar
  e rejeita novas entradas. Os clientes MusicBrainz e de imagem propagam contexto,
  limitam respostas JSON a 8 MiB e usam timeout de 10 segundos. Falhas de transporte
  não incluem URL com credenciais no log testado.
- Download automático de imagem respeita cancelamento, limite de 10 MiB, timeout
  de 10 segundos e até três redirecionamentos; URLs HTTP(S) próprias continuam
  aceitas. Gravação temporária com renomeação evita expor arquivo parcialmente
  escrito. O fallback de imagem escreve apenas uma resposta por requisição.
- Revisão seguida de nova escuta invalida confirmação de merge e de exclusão,
  preservando a escuta; falha de commit por chave estrangeira não deixou escrita
  parcial. O teste WAL existente valida conflito de snapshot com outra conexão.
- A revogação impede nova autenticação; requisição já autorizada pode terminar.
  Na exclusão do perfil antes do salvamento, o teste de barreira confirmou erro
  na gravação, nenhuma escuta no outro perfil e rejeição da chave removida.
  A importação cancelada após o primeiro salvamento informa resultado parcial e
  limpa o arquivo temporário; não implica rollback das escutas já gravadas.
- Exportação com mais de 1000 escutas e duas inserções concorrentes confirmou a
  paginação dinâmica: uma inserção antes do cursor não aparece e outra posterior
  aparece. Cancelamento do writer deixa stream incompleto, sem sucesso fictício;
  a exportação não promete snapshot consistente.
- Relay recebeu 429 sem repetição e interrompeu espera de retry 5xx ao cancelar.
  Em processo real, com relay local em andamento, SIGINT resultou em saída 0 em
  aproximadamente 0,004 s e uma escuta íntegra no SQLite. Isso não garante entrega:
  o relay continua best-effort, sem fila durável ou drenagem no encerramento.

## Comandos e resultados

- `GOCACHE=/private/tmp/koito-go-cache GOPROXY=off go test ./...`: passou.
- `go test -race ./queue ./internal/imagefetch ./imagecache ./internal/images ./internal/relay ./internal/catalog ./internal/db/sqlite ./internal/export ./engine`: passou; os novos casos de exclusão de catálogo e perfil foram executados separadamente com `-race`.
- `go vet` nos pacotes alterados: passou.
- `python3 scripts/validate_multiuser.py --binary /private/tmp/koito-issue020-final`:
  passou nas bases nova e migrada; integridade SQLite preservada.
- `python3 scripts/validate_admin_webui.py --binary /private/tmp/koito-issue020-final --work-dir /private/tmp/koito-020-admin-validation`:
  passou, incluindo cinco formatos de importação e exportação/round-trip.
- `python3 scripts/validate_isolation_019.py --binary /private/tmp/koito-issue020-final`:
  passou nos modos Go e React/API.
- `python3 scripts/validate_shutdown_020.py --binary /private/tmp/koito-issue020-final`:
  passou no cenário de relay em andamento acima.

Medição local de `TestListenBrainzBodyLimit`, com três corpos de teste no limite,
levou 1,36 s e registrou pico RSS de 139.034.624 bytes para todo o processo de
teste, incluindo fixture e strings. Não é benchmark de produção nem limite de
memória por requisição.

## Limites e trabalhos pendentes

A cobertura ainda não demonstra todas as ordens possíveis entre ingestão, exclusão
ou mesclagem de catálogo e exclusão de perfil, nem o reinício durante cada fase de
importação. Falhas de disco/limpeza e o comportamento de seguidores concorrentes
no fallback de imagem não foram integralmente injetados nesta execução. O limite
por bytes não é uma garantia de teto de memória do parser. A compressão nativa de
imagem já iniciada não é interrompida pelo contexto; funções legadas de fila que
ignoram contexto podem continuar após shutdown. Goroutines de inicialização e
relay não são drenadas por uma política geral de encerramento. O aceite transversal,
incluindo reexecução da matriz completa e parecer consolidado, permanece na 021.

## Revisão Advisor

Sol, esforço alto: **aprovado com ressalvas**. Não identificou falha crítica
pendente no código revisado. Ressaltou a natureza parcial da importação, a
exportação dinâmica, o relay best-effort, a autorização já concedida e os limites
do cancelamento cooperativo e do processamento de imagens. Confirmou testes
selecionados com `-race`; as demais execuções acima foram realizadas pelo agente
principal. A aprovação não substitui os cenários pendentes registrados aqui.
