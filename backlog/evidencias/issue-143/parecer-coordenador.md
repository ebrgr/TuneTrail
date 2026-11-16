# ISSUE-143 — Parecer do coordenador (Claude Sonnet, esforço médio)

Revisão do diff da branch (commit `aee2f15`, depois reaplicado sobre a v0.23) feita pelo coordenador, por decisão do usuário; sem Reviewer externo e sem Advisor.

**Parecer: APROVADO, sem itens obrigatórios.**

## Operação destrutiva
- **Escopo:** apaga só linhas de `listens` com `profile_id` do perfil da rota, dentro de uma transação; a seleção usada para a contagem mostrada e para a exclusão é a mesma consulta (`profileDeleteSelection`), e a exclusão revalida a impressão digital do plano e a contagem antes de apagar (`ApplyProfileDelete`); divergência vira conflito, sem apagar parcialmente. O catálogo compartilhado e outros perfis não são tocados.
- **Regras (decisões do usuário):** faixa e álbum = título e conjunto completo de nomes principais de artistas, igualdade binária, sem ordem (comparação de conjuntos por `EXCEPT` nos dois sentidos); artista = escutas de faixas em que ele é o único crédito, colaborações preservadas. Sem aliases e sem normalização de caixa.
- **Injeção:** o SQL só concatena nomes fixos de tabelas escolhidos por um `switch` sobre o tipo; todo valor do usuário vai por parâmetro; tipo inválido retorna erro.
- **Autorização:** o plano vive em memória por 10 minutos, preso a sessão, perfil, rota e tipo; confirmação exige CSRF e ticket de uso único, consumido mesmo em falha; sem repetição automática.
- **Destino:** página inicial do perfil (`/{slug}`) para faixa, álbum e artista.
- **Causa de "clicar não apaga nada":** `form.action` era encoberto por um campo `action` dos formulários antigos; o diálogo agora lê `getAttribute('action')`.

## Observações (não bloqueiam)
1. Com contagem 0 a janela ainda mostra o botão de excluir; nada é apagado e a contagem aparece, mas poderia desabilitar o botão.
2. O código da exclusão global do catálogo (revisão e confirmação) continua no servidor sem alcance pela interface, apenas por plano antigo; vale removê-lo numa tarefa de limpeza.
3. Depois do rebase não foram repetidas as validações em Chrome (decisão do usuário).
