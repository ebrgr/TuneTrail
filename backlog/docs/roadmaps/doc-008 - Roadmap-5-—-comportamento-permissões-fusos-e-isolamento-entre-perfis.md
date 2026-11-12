---
id: doc-008
title: Roadmap 5 — comportamento, permissões, fusos e isolamento entre perfis
type: specification
created_date: '2026-11-12T23:39:00Z'
updated_date: '2026-11-12T23:39:00Z'
tags:
  - roadmap
---
> Importado de `docs-dev/ROADMAP_05_VALIDACAO.md`. Original removido em 12/11/2026; consulte o histórico do Git.

# Roadmap 5 — comportamento, permissões, fusos e isolamento entre perfis

Estudo de 05/11/2026 sobre a árvore de trabalho local, incluindo entregas ainda
não publicadas. Issues **001–015 fechadas e resolvidas por aceite explícito do
usuário** nesta data. Este documento planeja a próxima etapa; não afirma que sua
matriz já foi executada. Revisão com [Advisor](../../../.agents/advisor.md), Sol, esforço alto.

Base vigente: **SQLite com esquema consolidado v3**. Não alterar migrações 001–003
nem usar o banco operacional nos experimentos. React continua padrão; Go é
opcional (`KOITO_WEB_UI=go`). Remoção de React e aceite visual geral pertencem à
parte 6. Esta etapa não cria contas privadas por perfil: há administrador único,
perfis públicos e chaves pessoais de ingestão.

## Objetivo e conclusão

Validar o sistema como um conjunto, atravessando HTML, JSON, ListenBrainz,
serviços, SQLite, cache e I/O. As etapas anteriores já possuem testes relevantes;
o trabalho é tornar explícita a cobertura, preencher lacunas e corrigir defeitos
reproduzidos. Igualdade Go/JSON não basta quando ambos compartilham o mesmo bug:
a fonte de verdade será uma fixture com resultados calculados independentemente.

Há atenção imediata em agregação por dia civil e seleção de períodos em fusos
fracionários/viradas de ano. Existem também limites assumidos a exercitar sob
concorrência: importação parcial, exportação dinâmica, relay best-effort e planos
de confirmação em memória. Eles não implicam, por si só, falha de segurança.

## Arquitetura e cobertura encontradas

| Área | Código e cobertura existente | Complemento necessário |
| --- | --- | --- |
| Seleção de perfil | `engine/middleware/profiles.go`, `internal/db` e testes `engine/profiles_backend_test.go` | Rota explícita, legado com zero/um/vários perfis, IDs estrangeiros e falha do store em toda família |
| Leitura pessoal | `internal/listening`, `internal/webui/*_test.go`, SQLite `profiles_test.go` | Oráculo independente para home, rankings, histórico, mídia, busca, atividade, interesse, summary e tocando agora |
| Escrita administrativa | `internal/admin`, `adminweb`, `adminui`, `engine/admin*_test.go` | Matriz por método/ator em servidor real nos dois modos, expiração, reinício, abas e requisições ambíguas |
| Tempo | `internal/webquery/*_test.go`, `internal/summary/timeframe_test.go`, `webui/graphs_test.go` | Fronteiras SQL reais, offsets 30/45 minutos, virada local/ISO e relógio controlado |
| Dados concorrentes | SQLite `mutation_plan_test.go`, `merge_test.go`, `profiles_test.go` | Interleavings determinados entre ingestão, revisão, exclusão, import/export e revogação |
| Integração | `scripts/validate_multiuser.py`, `validate_webui.py`, `validate_admin_webui.py` | Relatório único com casos, observáveis, cobertura negativa e limitações |
| I/O | `internal/importer`, `internal/export`, `internal/relay`, testes de imagens/avatars | Cancelamento, interrupção de stream, transporte lento, compensação e reinício controlados |

Os [resultados da 015](../../evidencias/roadmap-04/ISSUE_015.md) permanecem evidência
histórica, com 55 capturas e testes aprovados. Não substituem a matriz desta etapa.

## Contratos a preservar e decisões a tornar explícitas

| Superfície | Identidade e escopo | Invariante |
| --- | --- | --- |
| Leitura pública Go/JSON | Slug da rota; legado somente com um perfil | Resultados pessoais não incluem escutas de outro perfil; perfis são públicos, não confidenciais |
| Administração HTML/JSON | Sessão de administrador | Chave de player não concede privilégio; perfil selecionado não altera identidade do administrador |
| ListenBrainz | Chave vinculada ao perfil | Corpo, cookie ou query não reatribuem destino; chave revogada não autoriza novas requisições |
| Escuta/import/export/chave/relay/avatar | Perfil explícito | Uma aba permanece no seu destino; arquivo importado não redefine perfil; download é pessoal |
| Catálogo e imagens de mídia | Compartilhado globalmente | Edição/merge/delete podem afetar todos; isso deve ser informado, não confundido com vazamento |
| Busca global para gestão | Administrador autenticado | Busca pública continua pessoal; catálogo sem escutas pode ser selecionado somente na gestão autorizada |
| Tocando agora | Cache `now-playing:{profileID}` | Atualizar/invalidar um perfil não limpa nem substitui outro |
| Confirmações HTML | Sessão, prazo e plano/ticket | POST repetido não duplica mutação; GET/HEAD não executam action; reinício exige nova revisão |

HTML possui CSRF próprio; JSON mantém contrato existente com SameOrigin e sessão.
Não exigir token HTML silenciosamente no JSON nem mudar payloads dos players.
Revalidar autorização no servidor, inclusive downloads/HEAD, IDs e origem. Testar
400/401/403/404/409/422/429/500 conforme cada contrato, sem exigir um status único
para fluxos com redirect de login. Falha de banco não equivale a perfil inexistente
nem a contagem zero. `ProfileScope` atualmente retorna 404 para todo erro de lookup
por slug: distinguir ausência e infraestrutura merece regressão própria.

## Achados e hipóteses prioritárias

| Prioridade | Evidência atual | Encaminhamento |
| --- | --- | --- |
| Alta | Leitura: `sqlite/listen_activity.go` arredonda timestamps para hora UTC antes de converter em data local, em atividade/streaks | Reproduzir escutas antes/depois da meia-noite em Kathmandu e Lord Howe; validar total e distribuição por dia na 018 |
| Alta | Leitura: `GetActiveDays` agrupa por 86400 segundos UTC e usa `time.Now()` sem conversão antes de compor limite no fuso | Oráculo de dias civis; duas datas locais em um dia UTC e vice-versa; conferir média diária e escopo temporal na 018 |
| Alta | Leitura: atividade usa `< t2.Unix()` embora fim civil seja próximo de 23:59:59; helpers de intervalo usam `now.Year/Month/ISOWeek` sem converter `now` ao fuso solicitado | Testar último segundo e viradas de mês/ano/semana com relógio fixo, antes de definir correção |
| Média | Leitura: HTML usa validação própria; JSON legado aceita defaults; `HTML` usa `Query.Get`, enquanto rewind rejeita duplicados | Especificar casos válidos, inválidos e repetidos; diferenças deliberadas documentadas; não igualar parsers indiscriminadamente |
| Média | Leitura: `ReadStats` filtra contagens por timeframe, mas dias ativos/streaks recebem só fuso | Decidir/documentar significado all-time versus período por campo; não mudar média silenciosamente |
| Média | Limite documentado: export pagina dinamicamente, import pode gravar parcialmente | Testar concorrência e cancelamento contra contrato real, sem prometer snapshot/rollback integral |
| Média | Leitura: relay é goroutine com contexto independente e timeout; planos/tickets são locais ao processo e fingerprint global conservador | Testar término, retries, reinício e conflitos; não prometer entrega durável nem suporte distribuído |
| Média | Leitura: cookie Secure depende da requisição/proxy; cache administrativo no-store | Definir implantação/proxy confiável, verificar cabeçalhos e sigilo sem inventar suporte novo |

Os cinco casos temporais abaixo foram reproduzidos pelo Advisor em SQLite v3
em memória; os demais itens permanecem leitura/risco. Não foi reproduzida
vulnerabilidade nesta análise. Bugs funcionais encontrados na etapa 5
serão tratados nas novas issues, preservando o encerramento aceito de 001–015.

### Reproduções do Advisor — prioridade alta na 018

| Caso | Esperado | Observado |
| --- | --- | --- |
| Atividade Kathmandu:2024-05-01T00:05+05:45 | Dia01/05 | Dia30/04 |
| Atividade maio, com escuta adicional31/05 23:59:59 | Total2 | Total1 |
| Dias ativos30/04 23:55 e01/05 00:05 em Kathmandu | Dois dias civis | Um dia |
| Month1 sem ano, agora2024-12-31T23:30Z em Kathmandu | Janeiro2025 | Janeiro2024 |
| Week53 sem ano, agora2021-01-01 | Início2020-12-28 | Início2022-01-03 |

A execução chamou o store real em `sqlite.NewInMemory()` e o helper de intervalo,
sem rede ou banco operacional. Fonte/procedimento nas [evidências](../../evidencias/roadmap-05/README.md).
O Advisor aprovou o plano **com ressalvas**, exigindo que esses defeitos entrem
como regressões/correções novas na 018. São defeitos de comportamento existentes,
sem comprovação de falha de autenticação ou isolamento.

## Fixture e matriz de execução

Criar um cenário pequeno determinístico com perfis Alpha/Beta e Empty, catálogo
compartilhado e exclusivo, faixa sem escuta, múltiplos artistas, aliases Unicode,
duração zero, imagens compartilhadas, duplicata pessoal e mesmo segundo em duas
pessoas. Isolar cada caso que altera dados em uma cópia nova. Uma fixture ampliada
com mais de 1000 escutas deve cruzar páginas de export e empates por timestamp.
Registrar contagens por perfil e IDs sobreviventes antes/depois; verificar
`integrity_check` e `foreign_key_check`. Não derivar esperado do próprio handler.

Eixos obrigatórios:

- Modo Go e React/API; rotas canônicas, aliases legados, métodos inesperados,
  assets/API fora do dispatcher e manifestos Go/TS sincronizados.
- Visitante, sessão válida, expirada/revogada, usuário não-admin sintético, cookie
  inválido/duplicado, chave A/B/revogada e combinações cookie+chave conflitantes.
- Perfis zero/um/vários; mudança de slug, exclusão, duas abas e navegação de retorno.
- UTC, America/Sao_Paulo (incluindo DST histórico), America/New_York,
  Asia/Kathmandu e Australia/Lord_Howe; fuso do processo diferente do solicitado.
- Precedência `KOITO_FORCE_TZ` > query válida > cookie válido > fuso do processo;
  query/cookie inválidos, aliases, parâmetros repetidos e limite entre dois dias.
- Ano bissexto, semana ISO 1/53, mês sem ano, último segundo, intervalo explícito,
  all-time, registros futuros/negativos suportados, períodos em andamento e DST
  inexistente/ambíguo em escuta manual. Diferenciar calendário de janela móvel;
  estabelecer o contrato de AddDate nos períodos móveis antes de mudá-lo.
- Revogação/exclusão simultânea: definir a fronteira entre nova requisição e
  trabalho previamente autorizado; não presumir cancelamento retroativo.
- Falhas de consulta, escrita, commit, disco, stream e transporte local; deadlines,
  ingestão simultânea e reinício durante operações selecionadas.

Para DST manual, rejeitar horário inexistente e exigir offset quando ambíguo;
confirmar o instante persistido e a apresentação local. O limite já documentado
de offsets históricos com segundos não passa a ser suportado nesta etapa sem
escopo específico. Relógio fixo/injetável deve ser mínimo e compatível com API
atual; não criar scheduler ou framework de teste genérico.

## Sequência de issues locais

1. [016 — contratos, fixture e matriz](<../../tasks/iss-016 - 016-—-Contratos-fixture-e-matriz-de-validação-transversal.md>).
2. [017 — permissões, sessões e origem](<../../tasks/iss-017 - 017-—-Validar-permissões-sessões-origem-e-formulários.md>).
3. [018 — fusos, intervalos e agregados](<../../tasks/iss-018 - 018-—-Corrigir-e-validar-fusos-intervalos-e-agregados-civis.md>).
4. [019 — isolamento, ingestão e cache](<../../tasks/iss-019 - 019-—-Validar-isolamento-entre-perfis-ingestão-e-cache.md>).
5. [020 — concorrência, falhas e I/O](<../../tasks/iss-020 - 020-—-Validar-concorrência-falhas-importação-exportação-e-I-O.md>).
6. [021 — integração e aceite transversal](<../../tasks/iss-021 - 021-—-Integrar-evidências-e-concluir-aceite-transversal-da-etapa-5.md>).

A 016 fixa critérios e compartilhamento de fixtures. As 017–020 podem reaproveitar
testes existentes, acrescentando somente casos necessários. A 021 reúne resultados
e pendências, solicita revisão Advisor e prepara a parte 6. Corrigir um defeito
exige primeiro regressão que falha no estado anterior; mudanças visíveis recebem
CHANGELOG. Não abrir migration ou alterar runtime somente para organizar o estudo.

## Evidências e porta de saída

Guardar procedimentos e resultados em [comparisons/roadmap-05](../../evidencias/roadmap-05/README.md).
Cada caso deve identificar cenário, modo, ator, perfil, fuso, relógio, operação,
status, observação no banco/cache, esperado independente e resultado. Registrar
seed, comando, versão de Go/Node quando executado, configuração sem segredos e
se o resultado veio de leitura, teste ou reprodução. Falhas/limitações permanecem
visíveis; nenhum relatório temporário com caminhos pessoais vira fonte única.

Na execução: `go test ./...`, race nos pacotes alterados/concorrentes, vet, cliente
typecheck/test/build e três validadores reais sobre bases descartáveis. Não repetir
suítes caras sem mudança relevante. Servidores externos simulados em loopback;
sem tokens pessoais, envio a serviços reais ou acesso ao banco operacional.

O aceite exige casos críticos aprovados, nenhuma fuga de autorização/isolamento,
regressões temporais corrigidas ou decisões explícitas, nenhuma corrupção,
servidores de teste encerrados e parecer Advisor registrado. Medir tempos/recursos
de cenários controlados informa limites, não certifica desempenho de produção.
Capturas funcionais só quando ajudam a comprovar retorno/erro/fuso; comparação
visual geral e remoção de React continuam reservadas à parte 6.

## Trabalho realizado neste estudo

Leitura dos roadmaps, relatório 015, rotas, middleware, consultas, intervalos,
serviços e testes. Fechamento local de 010–015 com sufixo `-solved.md` e atualização
de referências. Nenhuma implementação das novas issues e nenhuma publicação.
Resultados de execução e revisão estão no registro de evidências desta etapa.

## Atualização — issue 016

Fechada e resolvida por aceite do usuário em 05/11/2026. A [matriz](../../evidencias/roadmap-05/ISSUE_016_MATRIZ.md)
inventaria os contratos e aponta cobertura/lacunas de 017–021. A fixture pequena e
ampliada foi verificada em SQLite v3 descartável; não controla relógio do servidor.
Os defeitos temporais reproduzidos continuam para correção na 018.

## Atualização — issue 017

Fechada e resolvida por aceite do usuário em 05/11/2026. [Resultados e limites](../../evidencias/roadmap-05/ISSUE_017.md).
A matriz usa bancos e servidores descartáveis; o Advisor aprovou as correções.
As issues 018–021 continuam abertas.

## Atualização — issue 018

Fechada e resolvida por aceite do usuário em 05/11/2026. [Resultados, contratos preservados e
limites](../../evidencias/roadmap-05/ISSUE_018.md). As cinco reproduções foram corrigidas;
as issues 019–021 continuam abertas.

## Atualização — issue 019

Fechada e resolvida por aceite do usuário em 05/11/2026. [Resultados e limites](../../evidencias/roadmap-05/ISSUE_019.md).
Isolamento foi exercitado com dados fictícios nos modos Go e React/API; limites
de contexto interno e de gráficos permanecem registrados. Issues 020–021 abertas.

## Atualização — issue 020

Fechada e resolvida por aceite do usuário em 05/11/2026. [Resultados, correções e limitações](../../evidencias/roadmap-05/ISSUE_020.md).
O limite de ingestão e os fluxos de fila e imagem foram endurecidos; os cenários
concorrentes e de falha ainda não cobrem toda a matriz prevista na issue.

## Atualização — issue 021

Fechada e resolvida por aceite do usuário em 05/11/2026, com revisão final do Advisor aprovada com ressalvas. [Matriz transversal,
resultados e checklist da parte 6](../../evidencias/roadmap-05/ISSUE_021.md). A
issue 020 foi aceita com limitações registradas; as verificações não executadas
continuam identificadas no relatório. React permanece padrão e SQLite v3 é a base.
