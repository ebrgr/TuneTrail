# ISSUE-218 — instruções para retomar (registros de execução)

O usuário escolheu a opção 1 em 27/11/2026: **apagar os registros de sucesso quando o trabalho terminar**. Continuam valendo o prompt original e `INSTRUCOES-D.md`; o que muda está abaixo.

## O que fazer
- Quando o trabalho de enriquecimento chegar a um estado final, apague na mesma transação as linhas de `enrichment_provider_runs` com `state='found'` daquele recibo. Cubra todos os caminhos que encerram um trabalho, não só `closeEnrichment`.
- Só apague quando a atualização protegida pelo lease afetar o trabalho: um executor que perdeu o lease não pode apagar linhas de outro.
- Enquanto o trabalho não terminar, as linhas continuam, e a retomada segue pulando as fontes com resposta final. `ambiguous` e `failed` permanecem.
- Comente o código em inglês explicando por que as linhas de sucesso podem sair. Não edite a migração 025 nem limpe, na 026, linhas de trabalhos antigos.
- Testes: retomada de trabalho interrompido sem repetir a consulta; linhas `found` ausentes após o fim; `ambiguous` e `failed` preservadas; lease perdido não apaga nada. Prove quebrando o código.

## Medição
Meça de novo com o código final, incluindo a inserção e a remoção das linhas (sem `VACUUM` depois das remoções), para 1, 3 e 10 escutas por faixa, e atualize `MEDICAO.md`. Estimativa do coordenador: 671, 220 e 66 bytes por escuta. Pare e pergunte se o pior caso passar de cerca de 700.

## Exportação
A limitação de exportação virou a ISSUE-234. Não mude o formato nesta tarefa; cite a ISSUE-234 no HANDOFF.

## Entrega
Conclua o que o HANDOFF lista como pendente (critérios, ciclo de vida, truncados e ambíguos, migração, `make check`, CHANGELOG, graphify) e faça o commit.
