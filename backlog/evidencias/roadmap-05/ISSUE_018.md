# Issue 018 — fusos, intervalos e agregados civis

Implementada e aceita pelo usuário em 05/11/2026. [Issue](<../../tasks/issue-018 - 018-—-Corrigir-e-validar-fusos-intervalos-e-agregados-civis.md>) · [Roadmap 5](<../../docs/roadmaps/doc-008 - Roadmap-5-—-comportamento-permissões-fusos-e-isolamento-entre-perfis.md>).

## Correções

As cinco [reproduções originais](reproduce_time_results.txt) viraram regressões
que falharam antes da implementação. Agora atividade, sequência e dias ativos
agrupam os instantes originais em datas civis no fuso solicitado. O último segundo
do mês entra na consulta; mês implícito usa o ano local e semana implícita usa o
ano ISO. A retrospectiva usa o mesmo ano local para o denominador de dias.

Uma data civil é identificada por componentes de ano/mês/dia, independentemente
da meia-noite. A apresentação usa o primeiro instante existente no dia. Isso cobre
meia-noite inexistente em São Paulo/Asunción e repetida em Tunis; sequências
incluem a segunda ocorrência da última hora de 16/02/2019 em São Paulo. Os
instantes persistidos e o esquema SQLite v3 não mudaram.

O preenchimento de atividade usa 12 períodos por padrão, semanas ISO iniciadas
na segunda e incremento anual correto. JSON e HTML fixam um relógio por resposta;
a ausência de fuso preserva o fuso do processo. Datas de início/fim de calendário
são inclusivas, com precisão de segundos; a consulta de dias ativos e sequências
usa o início do próximo dia como limite exclusivo. Parâmetros explícitos de data,
tz e filtros seguem os contratos existentes.

## Validação

- Os testes `internal/db/*issue018_test.go` e `internal/db/sqlite/*issue018_test.go`
  cobrem as reproduções, último segundo e limites adjacentes, filtros por artista,
  álbum e faixa, perfis, timestamps pré-1970, Kathmandu, Lord Howe, UTC,
  New York e DST histórico de São Paulo. Um relógio explícito testa sequência e
  exclusão do próximo dia; Tunis testa as duas ocorrências da meia-noite.
- `internal/listening/activity_issue018_test.go` verifica padrões e dias
  preenchidos em novembro/2018; `internal/webui/graphs_test.go` verifica a célula
  do dia de meia-noite ausente. `timezone_issue018_test.go` compara o histórico
  HTML e JSON nos dois limites do mês usando dois registros conhecidos.
- Testes existentes de `internal/admin/listens_test.go` e jornada administrativa
  verificam escuta manual com hora inexistente e ambígua, offset e instante salvo;
  `internal/webquery` verifica ForceTZ, query, cookie, aliases e fallback local.
- `go test ./...`, `go test -race ./internal/db ./internal/db/sqlite
  ./internal/listening ./internal/summary ./internal/webui ./internal/webquery`,
  e `go vet` dos pacotes afetados passaram. Alguns resultados utilizaram cache;
  regressões específicas foram executadas com `-count=1`.
- `validate_webui.py` passou com servidores reais Go/React e bases equivalentes;
  `validate_admin_webui.py` passou jornadas administrativas, inclusive DST e
  verificação de integridade. As bases e servidores usados eram descartáveis.
- Advisor (Sol, esforço alto) revisou os cálculos e identificou testes adicionais
  para meia-noite repetida, fim do dia e relógio único; parecer final aprovado.

## Contratos preservados e limites

Dias ativos e maior sequência em `ReadStats` continuam históricos até o fim do
**dia local atual**; as contagens/duração são filtradas pelo período. A média
existente divide a contagem filtrada pelos dias ativos históricos. Não se anuncia
essa média como média do período. Essa semântica é uma decisão de compatibilidade,
não foi redesenhada nesta issue. Escutas futuras dentro do dia atual continuam
participando dos agregados de dias/seqüência, conforme o limite existente.

Períodos móveis preservam `AddDate` e o relógio original usado pelo helper;
calendário com ano omitido usa o fuso escolhido. No HTML, semana ISO inválida é
rejeitada; a API legada mantém sua normalização. `KOITO_FORCE_TZ` prevalece sobre
query válida, cookie válido e fuso do processo. Relógios compartilhados de
atividade não alteram as demais consultas de estatísticas.

A agregação SQL agrupa por segundo original e processa os resultados em fluxo:
memória Go cresce com dias, enquanto SQLite pode processar até um grupo por
segundo distinto. Não houve benchmark de bases grandes; carga e limites de
requisições de range permanecem na 020. O tratamento de horários históricos
com offset em segundos na entrada manual permanece limitado conforme a 015.

A mudança civil não certifica todos os casos da base IANA: uma data inteiramente
removida por mudança de linha internacional de data, como Pacific/Apia em
30/12/2011, não possui instante local. Hoje o preenchimento pode pular essa data
e o grid pode apresentá-la duplicada; sequência trata a lacuna como interrupção.
Definir se ela deve ser omitida ou representada como célula sem instante é trabalho
futuro. Nenhuma migration ou suporte universal a esses calendários é alegado.
