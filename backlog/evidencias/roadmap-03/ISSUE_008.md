# Issue 008 — retrospectiva Go e cálculos

**Situação atual: issue encerrada por aceite do usuário em 05/11/2026.**
O restante preserva o histórico de implementação e verificação.

O núcleo foi implementado usando o serviço `summary.GenerateSummary`, compartilhado
com a API, e SQLite com esquema consolidado v3. A rota pública permanece na
interface React até a integração da issue 009.

A seleção HTML usa a URL, valida ano/mês e fuso, aceita `month=0` como anual e
identifica o período corrente. Os controles usam links e formulário GET. A opção
`show_time` modifica a apresentação dos itens sem recalcular os totais. Erros de
consulta são distintos de um período vazio e os dados permanecem restritos ao perfil.

Os limites de consulta são resolvidos uma vez por resumo. As médias de calendário
usam os dias nominais do mês, ano ou semana; períodos móveis e intervalos explícitos
usam dias completos de 24 horas (mínimo um); `all_time` usa as datas civis desde a
primeira escuta do perfil até a data atual, inclusive. Nenhum filtro SQL herdado
foi alterado. Em fusos com meia-noite inexistente, como Assunção em outubro de
2017, o limite SQL continua sujeito à normalização histórica de `time.Date`; a
contagem do denominador usa os 31 dias nominais do mês. Mudanças nessa fronteira
exigem análise de compatibilidade separada.

## Evidências funcionais

- `go test ./...`: passou, incluindo os testes de servidor em porta local temporária.
- `go test -race ./internal/summary ./internal/webquery ./internal/webui` e
  `make ui.check`: passaram (cache Go em diretório temporário).
- Testes com SQLite v3 em memória e perfis distintos: limites de janeiro/2026,
  contagens pessoais de itens novos, duração conhecida, empate, limite top 5,
  ausência de dados, erros e comparação do mesmo resumo na API e no HTML.
- Testes de normalização e cálculo: fevereiro comum e bissexto, mudanças de DST,
  meia-noite inexistente, ano 1/9999, fusos, seleção de janeiro, intervalos
  explícitos, `all_time`, falha da primeira escuta e rejeição de período inválido.
- O Advisor revisou limites, cálculos e isolamento; a lacuna de testes de integração
  identificada na revisão foi coberta pelos testes SQLite adicionados em seguida.

A comparação visual e os pares de screenshots não foram feitos por solicitação do
usuário: ficam para o final da issue 009, junto com a ativação da rota e os testes
completos de navegação, recarga e voltar/avançar. A issue 008 continua aberta até
esse aceite; não há publicação, migration ou alteração do banco em operação.
