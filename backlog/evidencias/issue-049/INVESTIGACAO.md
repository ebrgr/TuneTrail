# ISSUE-049 — Investigação da associação residual

Data: 13/11/2026 (America/Sao_Paulo). Base: `bc2995e`, branch `main`, TuneTrail 0.8. Escopo: investigação, sem alteração funcional ou acesso ao banco do usuário.

## Atualização após implementação

O usuário escolheu explicitamente manter o esquema v3, sem migração. A proposta de migração abaixo é histórica e foi substituída pela implementação descrita em `ENTREGA.md`. Nenhuma migração, tabela ou trigger foi adicionada.

## Conclusão

Defeito confirmado no código atual e presente no código da tag local `v0.8`. Adicionar um artista à faixa cria duas relações; remover o artista elimina somente a relação da faixa. O álbum continua no ranking desse artista nos perfis que ouviram faixas do álbum. A integridade SQLite pode continuar válida apesar do erro de catálogo.

Não é possível distinguir retrospectivamente um crédito próprio do álbum de uma associação criada por faixa. Os dois históricos podem produzir exatamente os mesmos dados persistidos. Limpar tudo que não tenha faixa correspondente destruiria créditos legítimos de álbuns parciais. Apagar a última relação do álbum também pode excluir álbum, faixas e escutas de vários perfis.

## Contexto consultado

- `AGENTS.md`, skills graphify, issue-first e golang-master; `README.md`, `DEVELOPMENT.md`, `DEPLOYMENT.md`, `CHANGELOG.md` e guia de edição.
- Contrato do esquema consolidado (doc-018), revisão das pendências (doc-017), roadmap vigente (doc-032), ISSUE-010 e dependência da ISSUE-096.
- Registros das entregas recentes do Claude: ISSUE-062 (documentação/higiene), ISSUE-063 (suíte Python sobre o binário real incorporada ao `make check` e CI), ISSUE-064 (recuperação de migração/importação e rollback) e evidência `backlog/evidencias/recuperacao/ISSUE_064.md`. Seus aceites são históricos; esta investigação não reexecutou aquelas suítes.
- A ISSUE-080 planeja uma migração chamada v4. A numeração e o bootstrap precisam ser coordenados se a 049 introduzir uma migração antes dela.

## Causa e caminhos afetados

| Local | Constatação |
| --- | --- |
| `internal/admin/media.go:146` | `ChangeTrackArtist` chama `UpdateTrack` fora do escopo pessoal: edição do catálogo é global. |
| `internal/db/sqlite/track.go:307–312` | A adição grava `artist_tracks` e `artist_releases`, usando `INSERT OR IGNORE`. |
| `internal/db/sqlite/track.go:317–346` | A remoção apaga somente `artist_tracks`; preserva o crédito do álbum. |
| `internal/db/sqlite/album.go:465–484` | Ranking de álbuns por artista usa `artist_releases` e conta escutas das faixas do álbum, sem exigir relação do artista com a faixa. |
| `internal/webui/media.go:148–157` | Página do artista usa esse ranking para listar os álbuns. Conferência estática; não houve nova inspeção de navegador. |
| `db/migrations_sqlite/001_initial_schema.sql:26–31` | Relação tem somente artista, álbum e `is_primary`. Primariedade não registra procedência. |
| `db/migrations_sqlite/001_initial_schema.sql:113–122` | Trigger `trg_delete_orphan_releases` exclui o álbum ao remover seu último artista; as FKs propagam a exclusão às faixas e escutas. |

A transação de `UpdateTrack` já existe. O defeito é a ausência de representação da procedência e de reconciliação correspondente, não falta de atomicidade na edição atual. A proteção contra limpeza global de órfãos, introduzida nas correções anteriores, deve ser preservada.

Outros escritores a incluir no levantamento da implementação:

- `SaveAlbum` (`album.go:168`), `AddArtistsToAlbum` (`album.go:449`) e `mergeAlbumsIn` (`album.go:632`).
- `SaveTrack` (`track.go:136`), `AddArtistsToTrack` (`track.go:239`) e `mergeTracksIn` (`track.go:656`), além da exclusão e mudança de álbum de faixas.
- `SubmitListen` (`internal/catalog/catalog.go:94`, associação ao álbum antes de associar a faixa) e seus caminhos de importação. `AddArtistsToAlbum` não pode ser considerado automaticamente um crédito manual, pois a ingestão também o usa.
- Mesclagem/exclusão de artistas e consultas de associação por artista/título: a relação também participa da identidade do álbum, não apenas da apresentação.

## Reprodução executada

Arquivo: `observacao_test.go.txt`, carregado por overlay do Go como um teste do pacote SQLite. Usa `NewInMemory`, as migrações reais e a fixture existente com dois perfis. Nenhum arquivo `.go` da aplicação foi alterado.

| Teste | Resultado observado |
| --- | --- |
| `TestIssue049ObservationResidual` | Após adicionar/remover artista 2 da faixa 1, `artist_tracks` fica sem o vínculo e `artist_releases` mantém o vínculo. Álbum 1 continua no ranking do artista 2 em ambos os perfis, com uma escuta por perfil. |
| `TestIssue049ObservationAmbiguousOrigin` | Crédito de álbum previamente adicionado + edição de faixa e edição de faixa isolada terminam com snapshots idênticos de catálogo e histórico. |
| `TestIssue049ObservationLastCreditDeletion` | Ao tornar o vínculo residual o último crédito do álbum fictício e apagá-lo, desaparecem álbum 1, faixa 1 e as duas escutas dos perfis. Álbum/faixa 2 e suas escutas sobrevivem. |

Os três testes de observação passaram: eles afirmam o comportamento defeituoso atual e o risco da limpeza, não o comportamento desejado após uma correção. `checkIntegrity` passou nos cenários residual e de cascata, mostrando que integridade referencial não detecta esse defeito semântico.

Também passaram os testes existentes `TestTrackArtistUpdateUsesOneTransaction`, `TestPrimaryArtistRequiresRelation` e `TestRemoveArtistDoesNotPurgeUnplayedCatalog`. Total: **6 testes aprovados**, pacote concluído em 0,408 s. O teste transacional existente injeta falha na associação ao álbum e verifica rollback da associação à faixa.

Para repetir a partir da raiz do repositório:

```sh
python3 - <<'PY'
import json
from pathlib import Path
root = Path.cwd()
overlay = {'Replace': {
    str(root / 'internal/db/sqlite/issue049_observacao_test.go'):
    str(root / 'backlog/evidencias/issue-049/observacao_test.go.txt')
}}
Path('/tmp/tunetrail-issue049-overlay.json').write_text(json.dumps(overlay))
PY
CGO_ENABLED=0 go test -overlay /tmp/tunetrail-issue049-overlay.json ./internal/db/sqlite \
  -run 'TestIssue049Observation|TestTrackArtistUpdateUsesOneTransaction|TestRemoveArtistDoesNotPurgeUnplayedCatalog|TestPrimaryArtistRequiresRelation' \
  -count=1 -v
```

## Direção proposta para implementação

1. Criar migração aditiva posterior à v3 consolidada, preservando 001–003. Preservar vínculos existentes como legados protegidos: nem mesmo a existência atual de uma faixa permite inferir que o crédito não era também independente.
2. Representar justificativas coexistentes: crédito próprio, legado ambíguo e derivação das faixas. Uma origem única em enum não atende origem dupla. A representação concreta ainda precisa ser especificada.
3. Reconciliar derivações na mesma transação da mutação de faixas. Remover a justificativa derivada apenas quando nenhuma outra faixa do mesmo artista a sustentar; manter créditos próprios, primários e legados. Considerar criação, remoção, troca de álbum, exclusão e merges, além da ingestão e importação.
4. Tratar a trigger de órfãos explicitamente, sem desativar FKs. A edição de créditos não pode produzir cascata destrutiva. Revisar também os planos existentes de impacto de exclusão/mesclagem caso o contrato da trigger mude.
5. Oferecer revisão pontual dos vínculos históricos, com origem ambígua visível, impacto global e confirmação atualizada antes da remoção. O exemplo original Engenheiros do Hawaii / Beatles consta da issue; não foi reinspecionado no banco real nesta sessão.
6. Atualizar `schemaVersion`, `checkDatabase` e os caminhos do bootstrap: hoje recusam versões acima de 3, exigem versão 3 e aceitam legado apenas v2/v3 (`bootstrap.go:124,159,257,339`). Apenas adicionar SQL de migração não basta. Coordenar numeração, testes e descrição da ISSUE-080.
7. Reutilizar a infraestrutura de recuperação da ISSUE-064 para validar backup, interrupção, rollback transacional e restauração do novo upgrade. Não prometer downgrade do esquema: recuperação exige backup anterior completo e tratamento das escutas posteriores.

Decisões a fechar no contrato: significado de crédito explícito durante ingestão versus edição; como registrar promoção a primário; como revisar/remover legado com segurança; ranking de crédito próprio sem faixa correspondente. Preservar inicialmente o contrato atual de ranking, salvo decisão expressa em contrário.

## Revisão do Advisor

@Sol, effort low, revisou o diagnóstico e a direção de persistência CRUD em modo somente leitura. Parecer: **REPROVADO COM RESSALVAS** para considerar o desenho pronto para implementação. Confirmou a causa e exigiu justificativas coexistentes, preservação dos legados, reconciliação transacional e coordenação da migração. Recomendou testes de várias faixas do mesmo artista e idempotência. As decisões acima permanecem abertas; não houve aprovação de um patch ou esquema concreto.

## Cobertura ainda necessária na correção

Origem dupla nas duas ordens de criação; primários e promoção explícita; várias faixas do artista no mesmo álbum; álbuns parciais; dados legados ambíguos; operações repetidas; merges e exclusões; ingestão concorrente com edição; falha/rollback; migração e restauração; páginas e API; rankings com contexto de perfil; planos de impacto e integridade SQLite. A suíte completa `make check` e o e2e devem rodar após a implementação. Nesta investigação foram executados apenas os seis testes dirigidos descritos acima.

ISSUE-049 permanece **Aberta**. Nenhum critério de correção foi marcado como concluído; não houve alteração de runtime, migração, changelog, commit ou publicação.
