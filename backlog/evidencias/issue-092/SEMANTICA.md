# ISSUE-092 — Resumo para decisão

O [doc-044](../../docs/estudos/doc-044%20-%20ISSUE-092-—-Semântica-das-estatísticas-e-rankings.md) registra as fórmulas e referências da v0.47. Com 60 escutas no mês, 3 dias ativos no mês e 120 no histórico, a API atual mostra 0,50 por dia. Em um mês de 30 dias, a média seria 2,00 por dia civil ou 20,00 por dia ativo do próprio mês.

- **Média principal:** dias civis do recorte, incluindo datas com zero. Mostrar a média por dia ativo separadamente se for desejada.
- **Período em andamento:** um relógio e um fuso IANA para toda a consulta, limitar os eventos ao instante atual e incluir a data de hoje no denominador. Rotular «até agora» e preservar a distinção entre calendário e janela móvel.
- **Ranking da ISSUE-135:** padrão por gravação com alternância para faixas do lançamento, como já registrado na tarefa. Preferência persistente por perfil é uma decisão independente.
- **Eventos aceitos:** uma escuta por evento vivo do perfil, incluindo ocorrências autônomas. Usar «Sem lançamento» para reconciliar os totais. Recibos pendentes ou repetidos não acrescentam escutas. A v0.47 já soma o evento autônomo em reproduções, mas as junções legadas podem omiti-lo do histórico e do top de faixas.

## ESTADO

- Feito: inventário e recomendação no doc-044.
- Pendente: aceite do usuário para o AC1. Medições e propostas de correção em BENCHMARK.md.
