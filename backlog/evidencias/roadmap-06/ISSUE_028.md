# Issue 028 — helper React independente de perfis

Em 05/11/2026, a rota global `/theme-helper` foi corrigida para usar uma prévia
estática com conteúdo fictício, sem selecionar perfil, consultar histórico ou
gerar links para itens reais. O contrato estrito de `useProfile` foi preservado.
Go continua padrão; a rota existe apenas no React explicitamente selecionado.

Ao investigar a jornada, foram corrigidos também dois erros de tema personalizado:
o provedor consultava uma paleta inexistente após aplicar `custom` e perdia o
nome selecionado no reload. O editor valida as 14 cores CSS, rejeita JSON inválido
com mensagem acessível e preserva o tema anterior. Paletas armazenadas inválidas
retornam ao padrão; a propriedade legada `name` é ignorada.

## Validação

- Antes da correção, o navegador não encontrou o editor dentro de 12 segundos.
  Depois, `validate_theme_helper_028.py` e seu script Playwright passaram em
  servidores React locais com SQLite v3 descartável e 0, 1 e 3 perfis. Conferiram
  entrada direta, prévia fictícia, alteração e aplicação da cor, recarga,
  rejeição de JSON malformado, null, array, objeto incompleto e cor inválida,
  troca para Yuu, retorno ao padrão com armazenamento inválido e ausência de
  requisições de perfil ou histórico na prévia. Integridade e contagens ficaram
  preservadas. Recursos externos foram bloqueados; erros esperados de rede e
  autenticação do shell não foram tratados como exceções de JavaScript.
- `validate_migration_025.py` repetiu quatro migrações fictícias, backups,
  jornadas e integridade; em base migrada, o navegador encontrou o editor
  (`THEME_HELPER=editor-visible`). Este teste não exercitou a edição após migração.
- `npm run typecheck`, `npm test` (10 testes) e `npm run build` passaram. A
  correção altera apenas o cliente React; o backend de testes é o mesmo da 026.
- [Advisor](../../../.agents/advisor.md), Sol esforço médio, revisou e aprovou a
  implementação por leitura. A recomendação de ampliar a detecção de consultas
  ao histórico foi incorporada ao teste de navegador.

O visual da prévia foi simplificado de propósito e usa dados de exemplo; não é uma
comparação visual de todas as telas ou uma auditoria de acessibilidade. Permanecem
as demais ressalvas da [026](ISSUE_026.md), posteriormente aceita pelo usuário.
