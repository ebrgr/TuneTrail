# ISSUE-049 — Correção sem migração

Data: 13/11/2026. Branch: `fix/issue-049-associacao-artista-album`.

## Decisão e comportamento

O usuário escolheu manter o esquema SQLite consolidado v3. Nenhum arquivo de migração, tabela, view persistida, trigger, bootstrap ou versão do banco foi alterado.

A expressão SQL compartilhada `albumArtistRelations` combina os créditos próprios/legados em `artist_releases` com a participação atual em `artist_tracks JOIN tracks`. Pares já persistidos não são repetidos no ramo derivado: artistas e escutas não são contados duas vezes.

`SaveTrack` e `UpdateTrack` deixam de gravar créditos independentes de álbum. A ingestão identifica sua associação como derivada e a persistência usa as faixas como origem desse vínculo. Criar um álbum continua salvando seus créditos próprios. Os leitores de identidade por artista/títulos, detalhes, artistas de álbum e rankings usam a relação efetiva. Remover a última participação de faixa faz o álbum desaparecer do ranking daquele artista sem disparar o trigger de remoção de álbum.

Crédito independente, primário, legado ambíguo ou sustentado por outra faixa permanece. Escolher explicitamente um participante como artista primário do álbum materializa um crédito próprio na mesma transação. Merges de álbum copiam só créditos persistidos; a participação segue as faixas movidas. Merges de faixa mantêm a regra anterior de transferência de créditos para o álbum destino. As prévias mostram os artistas efetivos, mas conservam o cálculo destrutivo com os créditos persistidos, correspondendo ao trigger v3 inalterado. Foi removida uma rotina global sem chamadas que apagaria créditos independentes se fosse reutilizada.

## Dados antigos e recuperação

A escolha sem migração não identifica a origem de vínculos antigos. O caso histórico registrado na issue pode continuar visível até revisão pontual. **Não foi feita limpeza no banco real do usuário.**

O guia `docs/src/content/docs/guides/editing.md` explica o comportamento e oferece um procedimento offline para um único par artista/álbum, após revisão humana e backup completo. O SQL recusa crédito primário, participação sustentada por faixa e o último crédito persistido. As guardas evitam a cascata antiga; não devem ser retiradas quando um reparo for recusado. O procedimento e suas recusas foram exercitados em fixtures fictícias. Para desfazer um reparo confirmado, restaurar o backup com o aplicativo parado, considerando as escutas posteriores.

## Validação

- `make check`: aprovado após as alterações finais; `go vet`, suíte Go, verificações JavaScript, build e **10/10 jornadas e2e**.
- Testes novos em `internal/db/sqlite/album_artists_test.go`: ranking nos dois perfis, identidade por artista/título, esquema v3, créditos próprios nas duas ordens, legado, promoção a primário, outra faixa sem escutas, ausência de dupla contagem, preservação do álbum e histórico, ingestão repetida, merge de álbum, rollback de edição mista, criação/exclusão de faixa e guardas do reparo manual.
- `TestArtistAlbumsFollowTrackEdits`: páginas do artista e rankings HTML dos dois perfis mostram e removem o álbum conforme a participação. O artista sem escutas próprias no perfil volta a 404, preservando isolamento.
- Teste transacional existente atualizado para injetar erro na tabela realmente escrita (`artist_tracks`); testes anteriores de integridade, falhas, migração, recuperação e planos de impacto passaram na suíte completa.
- `TestPlanAccountsForArtistAlbumTrigger` mantém cobertura da prévia destrutiva no v3: outro artista derivado não impede a cascata de uma exclusão explícita do único artista persistido; impacto previsto corresponde à execução.
- `git diff --check`: aprovado. Diff revisado conforme a skill deslop.
- Advisor @Sol (effort low): **APROVADO. Nenhuma objeção ao design proposto.** Revisou preservação do legado, deduplicação, promoção explícita, escritores, leitores, merges e guardas da revisão pontual.
- Grafo atualizado pelo script AST-only do projeto: 2770 nós / 14536 arestas. O extrator não produz símbolos para o arquivo que contém somente a constante SQL; a execução terminou com sucesso.

A investigação anterior e `observacao_test.go.txt` documentam o defeito na base 0.8 antes da correção; o teste histórico de observação não é um teste da implementação corrigida. Não incluímos uma migração v4 e não é necessário renumerar a ISSUE-080.

## Estado da entrega

Implementada e verificada, aguardando aceite do usuário. Sem publicação, push, alteração do banco real ou limpeza global. Alterações simultâneas de outras tarefas foram preservadas fora do escopo desta entrega.
