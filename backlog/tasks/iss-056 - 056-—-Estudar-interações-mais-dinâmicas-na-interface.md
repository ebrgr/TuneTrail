---
id: ISS-056
title: 056 — Estudar interações mais dinâmicas na interface
status: Aberta
assignee: []
created_date: '2026-11-12T23:39:00Z'
labels:
  - ui
milestone: m-8
dependencies: []
references: []
type: spike
ordinal: 56000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
> Importada de `docs-dev/issues/056-estudo-interacoes-ui-dinamica.md` — numeração local 056. Original removido em 12/11/2026; consulte o histórico do Git.

**Status:** Aberta — análise futura · **Criada:** 10/11/2026 · **Roadmap:** [interações da interface](<../docs/roadmaps/doc-001 - ROADMAP.md>)

## Objetivo

Avaliar melhorias de interação para tornar o TuneTrail mais fluido, mantendo Go responsável pelos dados e pela renderização inicial e utilizando JavaScript de forma proporcional a cada necessidade. Esta issue registra recomendações para estudo: não autoriza por si só a implementação nem publica uma issue no GitHub.

## Situação atual

A interface da aplicação usa aproximadamente 197 linhas e 14 KB de código-fonte JavaScript, distribuídos em quatro arquivos; a contagem é indicativa, pois há várias instruções em uma mesma linha.

- [theme.js](../../internal/webui/static/theme.js): aplicação e persistência local de temas e paletas.
- [ui.js](../../internal/webui/static/ui.js): diálogos, abas, atalhos, temas personalizados e tratamento de falhas em imagens.
- [live.js](../../internal/webui/static/live.js): fuso, início da semana, período selecionado e exclusão rápida de escutas por administradores.
- [admin.js](../../internal/webui/static/admin.js): diálogo administrativo, copiar, arrastar arquivos e prevenção de envios repetidos.

O backend gera as páginas e as principais jornadas usam links e formulários. Os scripts em `scripts/` são usados em testes e capturas; a documentação Astro em `docs/` é um projeto separado. Sem JavaScript, permanecem alternativas para navegação, busca, filtros, idioma, gerenciamento e envio de arquivos; personalização visual e outras conveniências dependem de JavaScript.

## Recomendações para avaliação

As prioridades e complexidades abaixo são estimativas iniciais, sujeitas à análise das telas e contratos existentes.

| Proposta | Benefício | Complexidade estimada | Pontos a analisar |
| --- | --- | --- | --- |
| Atualização de “tocando agora” e escutas recentes | Exibir novidades sem recarregar a página inteira. | Média | Frequência, pausa quando a página não estiver visível, falhas, tráfego, isolamento de perfil e uso da API existente. Comparar polling com alternativas antes de decidir. |
| Busca com sugestões | Encontrar artistas, álbuns e faixas enquanto se digita, inclusive em edição e mesclagem. | Média | Reutilizar buscas existentes, limitar e cancelar requisições, resultados vazios, teclado, leitor de tela e dados por perfil. |
| Edição de tags e gêneros | Adicionar e remover tags com retorno imediato. | Média, após a persistência | Depende do modelo e das regras de origem da [issue 055](<iss-055 - 055-—-Aproveitar-tags-e-datas-de-lançamento-das-músicas.md>); distinguir gênero de tag livre e preservar alterações manuais. |
| Progresso de importação | Mostrar processamento, resultados parciais e erros de arquivos grandes. | Alta | Exige estudo de trabalho em segundo plano e acompanhamento no servidor; definir reinício, falha, duplicação, permissões e retenção de estado. |
| Filtros e paginação dinâmicos | Atualizar rankings e histórico com menos recarregamentos. | Média a alta | URLs compartilháveis, histórico do navegador, foco, indicadores de carregamento e equivalência com os filtros atuais. |
| Gráficos interativos | Consultar detalhes de pontos e intervalos diretamente no gráfico. | Média | Preservar tabelas acessíveis, limites corretos, uso por teclado e comportamento em telas pequenas. |

Ordem sugerida para investigação: começar por **tocando agora** e **busca com sugestões**, pois têm benefício visível e podem ser implementadas pontualmente. Para tags e gêneros, aguardar a definição do armazenamento e da procedência na issue 055.

## Diretrizes para o estudo

- Manter dados e cálculos no servidor e evitar introduzir um framework sem justificativa comparada com o código existente.
- Preservar caminhos funcionais com HTML e formulários quando fizer sentido; registrar claramente qualquer recurso que dependa de JavaScript.
- Manter autenticação, autorização, proteção contra requisições indevidas, isolamento por perfil e validação no servidor. Não confiar apenas em controles ocultos no navegador.
- Planejar mensagens traduzidas, estados de carregamento, erro e vazio, acessibilidade, navegação por teclado, temas, celular e conexão intermitente.
- Medir impacto em tráfego, desempenho e manutenção; evitar atualização ou requisições desnecessárias.
- Para cada proposta escolhida, descrever tela, comportamento, contrato com o backend, dependências, estimativa mais precisa e testes. Criar issues de implementação separadas se o escopo justificar.

## Critérios para concluir a análise

- [ ] Comparar as seis propostas, suas prioridades e alternativas, com estimativas revisadas.
- [ ] Definir quais fluxos permanecem utilizáveis sem JavaScript e como tratar falhas de rede e estados de carregamento.
- [ ] Registrar requisitos de acessibilidade, internacionalização, segurança, isolamento de perfis e desempenho.
- [ ] Identificar APIs e mudanças de backend necessárias; verificar dependência da issue 055 para tags.
- [ ] Propor sequência de implementação e issues próprias quando necessário, com capturas ou protótipos se ajudarem na decisão.
- [ ] Apresentar o estudo para avaliação do usuário antes de encerrar esta issue.
<!-- SECTION:DESCRIPTION:END -->
