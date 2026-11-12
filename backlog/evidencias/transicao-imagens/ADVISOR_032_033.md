# Revisão Advisor — issues 032 e 033

Data: 06/11/2026. Modelo Sol, esforço médio, persona
[Advisor](../../../.agents/advisor.md). Revisão solicitada pelo usuário antes da
etapa de distribuição. Nenhuma alteração de runtime foi feita nesta revisão.

## Parecer: REPROVADO COM RESSALVAS

Não foi encontrado bloqueador comprovado para iniciar a 034 nem falha crítica
na inspeção e nos testes direcionados. O parecer não equivale a aprovação
integral da integração ou dos builds de distribuição. As ressalvas são:

### P2 — Admissão não abrange todas as esperas e buffers

`internal/admin/artwork.go:103` lê até 10 MiB antes da espera em
`publicationMu` na linha 115. `internal/admin/profiles.go:144` e 155 aguardam
`profileMu` e a trava de publicação antes de entrar no pipeline na linha 175.
Essas Mutex não limitam o número de esperas nem permitem cancelá-las de imediato.

Cenário: manter uma publicação bloqueada e iniciar várias substituições de capa.
Elas podem reter um buffer por requisição sem receber `ErrBusy`, pois ainda não
entraram no semáforo. Cancelar o contexto não libera a espera na Mutex. Avatares
podem também bloquear outras operações que dependem de `profileMu`.

O limite de um trabalho ativo e quatro esperas aplica-se ao componente
`Transform`, não ao conjunto de requisições administrativas. A medição da 032
já excluía múltiplos buffers HTTP. Refinamento sugerido: admissão limitada e
cancelável antes da leitura/cópia, mantendo a proteção de publicação/associação.

### P2 — Contexto do primeiro pedido controla trabalho compartilhado

`imagecache/imagecache.go:119–131` e
`engine/handlers/image_handler.go:74–75,93–99` usam o contexto do primeiro pedido
no trabalho de `singleflight`.

Cenário: A inicia um derivado e B solicita o mesmo ID/tamanho. Se A desconectar,
o trabalho pode ser cancelado e B recebe erro mesmo estando conectado. O handler
pode retornar 500; na recuperação da source pode servir fallback. B também não
consegue cancelar independentemente sua espera em `Do`.

Não foi identificada corrupção persistente. A limitação está descrita na 033,
mas a afirmação de que ela apenas preservava a semântica anterior foi corrigida:
o antigo `GetImage` não propagava o contexto HTTP para a geração do derivado.
Refinamento sugerido: contexto/deadline próprio do trabalho compartilhado e
espera cancelável por participante, sem abandonar goroutines de codec.

### P3 — Guardas anteriores consomem erros específicos

`internal/admin/artwork.go:69–70,103–112` retorna `db.ErrInvalidInput` para excesso
de bytes antes do pipeline; nesses caminhos a resposta é 400, não 413. Formatos
que o sniff MIME não identifica como imagem, como SVG, recebem mensagem genérica
nessas guardas e em `internal/admin/profiles.go:168–169`, antes de `ErrFormat`.
Os mapeamentos específicos estão em `internal/adminweb/errors.go`.

Os limites permanecem efetivos. Padronizar erros tipados seria uma melhoria;
a documentação foi ajustada para distinguir rejeição no pipeline de rejeição
nas guardas anteriores.

### Risco histórico — publicação automática e coleta

`internal/catalog/associate_artists.go:137` publica antes de `SaveArtist` na linha
146; `internal/catalog/associate_album.go:136` publica antes de `SaveAlbum` na
linha 147. Esses produtores não usam `LockPublication`, ao contrário do coletor
em `internal/catalog/images.go:23`.

Coleta entre publicação e associação pode remover a source antes de sua
referência persistir. O handler tentará recuperá-la remotamente. É uma janela
histórica, sem regressão demonstrada do codec, registrada fora da correção das
032/033. A trava protege a sequência administrativa, não todas as publicações.

## Verificações sem objeção

Allowlist anterior à seleção genérica de decoder; limites de bytes, lados e
área antes do decode; revalidação posterior; correspondência de dimensões
VP8X/VP8/VP8L; rejeição de animação; orientação JPEG; checagens de contexto;
publicação temporária com rename; associação administrativa protegida até o
commit; limpeza em falha consulta referências sem contexto cancelado. Limitações
de ICC/CMYK e corpus estão explícitas e não provam fidelidade geral de cor.

## Validação executada pelo Advisor

Go 1.25.0, CGO desativado e GOPROXY offline: passaram pipeline, admin, adminweb e
handlers. Passaram os testes específicos da 033 de cache: legado, falhas,
múltiplos IDs, cancelamento de publicação e falha de escrita.

A suíte completa de cache parou no bloqueio de socket do sandbox em
`TestCanceledImageDownloadLeavesNoPublishedFile`; isso não representa falha do
produto. O Advisor não repetiu race, corpus externo, RSS, inspeção das capturas
ou builds distribuídos. Os resultados anteriores dessas verificações permanecem
nos relatórios das issues, com escopo próprio.

## Encaminhamento

A 034 pode prosseguir com essas limitações explícitas. Admissão administrativa e
contextos de `singleflight` precisam de demanda própria se forem exigidos fila
globalmente limitada e cancelamento independente. Esta revisão não implementou
essas melhorias nem validou antecipadamente a distribuição.

## Atualização posterior

Os três achados acionáveis foram tratados na [issue 036](ISSUE_036.md). O
registro acima preserva o parecer histórico, anterior às correções; ele não é
um novo parecer do Advisor. O risco histórico de coleta segue na issue 037.

A janela histórica de coleta foi tratada posteriormente na
[issue 037](ISSUE_037.md); o parecer acima é preservado como registro anterior.
