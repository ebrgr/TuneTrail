# ISSUE-218 — interrupção para decisão

## Feito

- Conferidos ISSUE-218 e ISSUE-217, doc-048, decision-011 e estudo complementar; consultados ai-memory e graphify.
- Registrado plano na tarefa e medida preliminar reproduzível em `MEDICAO.md` e `medir_metadados.py`.
- A medição estima 3.806,82 bytes por escuta para oito observações com os índices atuais. Parada conforme solicitação explícita.

## Decisão necessária

O limite informado é cerca de 500 bytes por escuta. É preciso decidir entre aceitar o custo maior ou estudar outra modelagem; mudar o vínculo das observações por escuta também exige decisão expressa. Não foi proposta nem implementada alteração de vínculo.

## Pendente e não verificado

Nenhum código ou migração foi alterado; todos os seis critérios permanecem abertos. Não foram verificados exportação, restauração, mesclagem ou exclusão dos tipos novos, nem executados os testes, `make check`, race, mutação de testes ou reconstrução do grafo. A medição é uma estimativa sintética isolada, não o tamanho final de um banco de aplicação. Sem commit, publicação ou alteração de status.
