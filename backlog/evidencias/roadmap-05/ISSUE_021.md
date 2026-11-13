# Issue 021 — integração e aceite transversal

Execução em 05/11/2026; [issue](<../../tasks/issue-021 - 021-—-Integrar-evidências-e-concluir-aceite-transversal-da-etapa-5.md>) ·
[matriz original](ISSUE_016_MATRIZ.md) · [roadmap](<../../docs/roadmaps/doc-008 - Roadmap-5-—-comportamento-permissões-fusos-e-isolamento-entre-perfis.md>).
As issues 016–020 foram fechadas por aceite explícito do usuário; os resultados e
limitações anteriores continuam registrados. A 021 foi fechada por aceite do usuário em 05/11/2026, com as ressalvas abaixo.

Ambiente: Go 1.26.3, Node v24.15.0, darwin/arm64, SQLite esquema v3. Credenciais
fictícias, imagens externas e MusicBrainz desativados; servidores externos simulados
em loopback. Não se usou o banco operacional. A fixture 016 usa timestamps fixos,
Alpha/Beta/Empty=3/2/0, UTC e referência 2025-06-01T12:00Z, mas não injeta esse
relógio no servidor. Os validadores web usam dados sintéticos e relógio real, com
escutas semeadas a partir de uma hora antes da execução; `KOITO_FORCE_TZ=UTC` é
usado na matriz de isolamento. Cada script informa sua própria configuração.

## Matriz caso → evidência → resultado

| Caso / modo, ator e escopo | Evidência executada | Resultado e alcance |
| --- | --- | --- |
| Família de rotas, métodos, atores e contratos | [Matriz 016](ISSUE_016_MATRIZ.md), manifestos Go/TS, testes Go e `npm run test` | Inventário histórico; não equivale a produto cartesiano completo. Testes de manifesto passaram. |
| Negativas administrativas, cookies, sessão, chave, origem, CSRF e logs; Go e React/API | [017](ISSUE_017.md), `validate_permissions_017.py` | 994 verificações negativas Go e 498 React/API; snapshots e integridade OK; marcadores fictícios ausentes dos logs testados. |
| Seleção 0/1/N, escopo pessoal e catálogo compartilhado; visitante, admin e chaves A/B; ambos modos | [019](ISSUE_019.md), `validate_isolation_019.py` | Contagens iniciais 3/2/0, após ingestão 4/2/0; rankings, busca, histórico, mídia, cache, duas abas, reinício e integridade verificados nos cenários do script. |
| Home, rankings e histórico; Go HTML versus JSON; React/API | `validate_webui.py` | 152 escutas, 52 na página 2 e totais de interesse artista/álbum/faixa 31/31/3; rotas Go e shell React testados. Não comprova cliques ou hidratação da SPA. |
| Gráficos visíveis sem JS: atividade e interesse; visitante Alpha/Beta/Empty, UTC | `TestRenderedGraphTablesMatchIndependentProfileCounts` em SQLite v3 | Soma HTML de atividade: 3/2/0; mídia compartilhada artista/álbum: 3/1 e faixa: 2/1; interesse tem os mesmos totais por entidade. Esperados vêm da fixture, não da API. A tabela de interesse por intervalo não foi comparada ponto a ponto com oráculo independente. |
| Cinco defeitos temporais: atividade, último segundo, dias ativos, mês e semana ISO | [Reprodução antes](reproduce_time_results.txt), regressões e [resultado 018](ISSUE_018.md) | Cinco falhas reproduzidas foram corrigidas; regressões Go passaram para os fusos/casos listados na 018. |
| DST manual, fuso e filtros | Testes 018, `validate_admin_webui.py`, `validate_webui.py` | Escuta manual ambígua/inexistente e limites testados; ForceTZ/query/cookie cobertos por testes dirigidos. Não são todas as zonas e datas IANA. |
| Login, criação/seleção, ações, retorno, import/export e reinício | `validate_multiuser.py`, `validate_admin_webui.py`, `validate_isolation_019.py` | HTTP real com perfis fictícios, JSON e formulários Go; cinco formatos, relay local, repetição e integridade. React/API é entrega de SPA e jornada API, sem clique em navegador. |
| Instalação nova e migração v2→v3 | `validate_multiuser.py` | Jornadas e `integrity_check` passaram em ambas. `validate_webui.py` usa banco v3 novo/cópia equivalente; `validate_admin_webui.py` usa banco Go novo. Estes dois não executaram sua própria jornada sobre base migrada. |
| Escopo, revogação e exclusão durante ingestão | Testes de barreira da [020](ISSUE_020.md) | Trabalho já autorizado pode concluir após revogação; exclusão do perfil antes do save causa erro, sem reatribuição ao outro perfil; nova chave revogada é negada. Não se cobre toda ordem possível. |
| Revisão, merge/delete, WAL e rollback | `concurrency_issue020_test.go`, `TestWALSnapshotRaceMapsToConflict`, testes anteriores de mutação | Save entre revisão e confirmação invalida o plano; teste de exclusão é sequencial. Falha de commit por FK diferida foi injetada em transação SQL direta, não em `ApplyCatalogMutation`; não cobre todo commit ou falha de disco. |
| Import parcial, export dinâmico, relay e encerramento | [020](ISSUE_020.md), testes de cancelamento e `validate_shutdown_020.py` | Importação pode deixar escutas já salvas; export não promete snapshot; relay é best-effort. Processo encerrou normalmente no cenário local, sem garantia de entrega. |
| Requisição ListenBrainz, bytes e cabeçalhos | `TestListenBrainzBodyLimit`, `TestListenDecoderStopsAtItemLimit`, `TestListenBrainzResponsesDeclareJSON` | 16 MiB, 1000 entradas por payload; regressão de Content-Type falhou antes da correção e passou em sucesso e falha de save. |
| Recursos, downloads e respostas em falha | [020](ISSUE_020.md), testes de fila/imagem | Limites, cancelamento cooperativo e escrita atômica exercitados; limites de compressão, jobs legados, falha de disco e interleavings não exaustivos permanecem na 020. |

## Execuções desta integração

Antes da correção pontual de cabeçalho: `GOCACHE=/private/tmp/koito-go-cache
GOPROXY=off go test ./...`, `go vet ./...` e `go test -race` dos pacotes de fila,
imagem, relay, catálogo, SQLite/db, export, cache, leitura, web, administração e
engine passaram; alguns resultados utilizaram cache. Depois da correção,
`go test ./engine ./engine/handlers ./internal/webui`, `go vet ./engine/...
./internal/webui/...` e testes dirigidos de cabeçalho e gráficos com `-race
-count=1` passaram. O teste de cabeçalho usa `ResponseRecorder.Result().Header`,
que captura os headers no momento de `WriteHeader`, e força falha de SQLite por
trigger descartável. `npm run typecheck`, `npm run test` (10 testes) e `npm run
build`, em `client`, passaram com dependências já instaladas.

Após reconstrução do binário final, `validate_multiuser.py`, `validate_webui.py`
e `validate_admin_webui.py` passaram novamente. `validate_permissions_017.py` e
`validate_isolation_019.py` passaram no binário anterior, antes da correção de
ordem do cabeçalho; a alteração não modifica escopo ou autorização. Os scripts
encerram seus servidores ao sair. Os diretórios de inspeção de web/admin são
cópias descartáveis em `/private/tmp`; não contêm credenciais reais. Não foram
produzidas capturas: esta verificação é numérica/funcional; comparação visual geral
está reservada à parte 6.

## Decisões e pendências, sem confundir com teste aprovado

- Estatística preservada: dias ativos e maior sequência são históricos até o fim
  do dia local atual, enquanto contagem/duração usam o período; a média existente
  combina essas grandezas. Janelas móveis preservam `AddDate`. Ver detalhes e
  casos de dias civis em [018](ISSUE_018.md).
- Perfis são públicos; catálogo/imagens são globais. Contextos internos sem perfil
  podem ler globalmente; isolamento público depende dos adaptadores de perfil. A
  cobertura observada não prova ausência de todos os caminhos possíveis.
- Uma data civil integralmente removida, como Apia 30/12/2011, e offsets manuais
  históricos em segundos continuam limitações documentadas na 018.
- Falta aplicar as jornadas web e administrativa a cópias individualmente
  migradas v2→v3; só a jornada multiusuário cobriu migração. Falta validação
  numérica independente de cada intervalo dos gráficos. Lacunas de falha real de
  disco, reinício de importação e ordens concorrentes estão detalhadas na 020.
- Testes desta etapa não executam hidratação, cliques, responsividade ou
  comparação visual completa em React. Checklist da parte 6 está separado abaixo.

## Checklist preparado para a parte 6

- [ ] Comparar Go e React em desktop/mobile, temas, fontes, sidebar e cards.
- [ ] Exercitar seleção de perfil, navegação e retorno, home/rankings/histórico,
      mídia e retrospectiva com JavaScript e no-JS quando aplicável.
- [ ] Conferir formulários administrativos, mensagens, estados vazios e erro,
      fusos, DST e gráficos; registrar screenshots reprodutíveis por cenário.
- [ ] Verificar acessibilidade, teclado, foco, contraste e layouts responsivos.
- [ ] Repetir jornadas de migração em cópias descartáveis; definir o destino das
      limitações técnicas ainda abertas antes da decisão de remover React.
- [ ] Manter fallback React até decisão explícita e aceite da parte 6.

## Revisão final Advisor

Sol, esforço alto: **aprovado com ressalvas**. Não identificou falha crítica nas
correções e evidências revisadas; confirmou por leitura a ordem dos cabeçalhos e
executou o teste dos gráficos. Sua execução independente do teste engine ficou
bloqueada pelo sandbox ao abrir porta; a execução do agente principal passou com
acesso local. O Advisor confirmou as lacunas de jornadas em bases migradas,
comparação por intervalo e interação visual da SPA, além das limitações anteriores.
O usuário aceitou o fechamento da issue em 05/11/2026, preservadas essas ressalvas.
