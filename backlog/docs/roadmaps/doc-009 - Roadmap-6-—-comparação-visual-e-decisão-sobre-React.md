---
id: doc-009
title: Roadmap 6 — comparação visual e decisão sobre React
type: specification
created_date: '2026-11-12T23:39:00Z'
updated_date: '2026-11-12T23:39:00Z'
tags:
  - roadmap
---
> Importado de `docs-dev/ROADMAP_06_COMPARACAO_VISUAL.md`. Original removido em 12/11/2026; consulte o histórico do Git.

# Roadmap 6 — comparação visual e decisão sobre React

Estudo de 05/11/2026 da árvore local, inclusive alterações ainda não publicadas.
As issues 001–021 foram encerradas pelo usuário. O [checklist da 021](../../evidencias/roadmap-05/ISSUE_021.md#checklist-preparado-para-a-parte-6)
é entrada obrigatória desta etapa. Revisão solicitada ao [Advisor](../../../.agents/advisor.md).

**Atualização:** o usuário aceitou a 026 e autorizou a retirada de React em 05/11/2026; a issue 030 registra esse trabalho. As referências a React abaixo são históricas.
O usuário determinou posteriormente Go como padrão na issue 027; React exige
`KOITO_WEB_UI=react` e não é fallback no modo Go. O esquema vigente é SQLite
consolidado v3; conservar migrações 001–003, dados e API ListenBrainz. Usar somente cópias SQLite e contas
fictícias, nunca o banco em operação.

## Situação observada

| Evidência | O que demonstra | O que permanece para a etapa 6 |
| --- | --- | --- |
| [Roadmap 1](../../evidencias/roadmap-01/README.md) e [2](../../evidencias/roadmap-02/README.md) | Prévia e pares visuais de navegação, home, rankings e histórico; diferenças já registradas | Retestar a árvore atual e estados/temas/viewports faltantes; capturas antigas são referência histórica |
| [Issue 009](../../evidencias/roadmap-03/ISSUE_009.md) | Pares de mídia/retrospectiva e jornadas parciais; divergências de gradiente, controles e gráfico SVG documentadas | Verificar árvore atual e ações restantes; gestão Go substituiu a ponte React descrita historicamente na 009 |
| [Issue 015](../../evidencias/roadmap-04/ISSUE_015.md) | 26 pares e três imagens de interação; ações Go, teclado e no-JS selecionados | Ampliar estados de erro, navegação integral, temas e acessibilidade; pares não provam igualdade pixel a pixel |
| [Issue 021](../../evidencias/roadmap-05/ISSUE_021.md) | Permissões, isolamento, fusos e somas HTML foram testados em cenários concretos | Hidratação/cliques React, jornadas web/admin em banco migrado e cada intervalo de interesse ainda não foram comprovados |

`engine/routes.go` monta o dispatcher Go só no modo `go`; caminhos não assumidos
caíam no servidor da SPA em `client/build/client` no momento do estudo. A issue
027 removeu esse fallback no modo Go; ele não era failover automático de uma página Go que falha. `/theme-helper` é rota React
explícita e está excluída do dispatcher Go no manifesto. Assets de páginas Go são
embutidos, porém `/public` usa `client/public`, `/favicon.svg` depende do build
React e templates usam Google Fonts;
verificar também fontes carregadas e indisponíveis. `Dockerfile`, `Makefile` e CI
ainda constroem ou copiam React. Remover apenas `client/` quebraria esses contratos;
o inventário completo e os destinos precisam ser decididos antes de qualquer retirada.

O repositório possui 12 paletas (`yuu`, `varia`, `midnight`, `catppuccin`,
`autumn`, `black`, `wine`, `pearl`, `rosebud`, `urim`, `match`, `lemon`). As
capturas anteriores privilegiam Yuu e Pearl; não afirmar validação visual dos 12
temas para todas as telas. O objetivo é equivalência funcional e visual avaliada,
não identidade pixel a pixel; diferenças intencionais exigem registro e aceite.

## Matriz de comparação planejada

| Família | Jornadas/estados prioritários | Verificações e pares mínimos |
| --- | --- | --- |
| Estrutura e perfis | Zero/um/vários, seleção, troca, retorno, links diretos, nome longo e imagem ausente | Desktop/celular, Yuu/Pearl, foco, sidebar, fontes, overflow e persistência de tema |
| Home, rankings, histórico | Dados e vazio; filtros, paginação, empate, busca, now-playing, voltar/avançar e duas abas | Conferir números e perfil com oráculo independente; capturar topo e região relevante abaixo da dobra |
| Artista/álbum/faixa | Relações múltiplas, compilação, capa ausente, interesse, atividade, histórico e acesso à gestão | Comparar valores por dia e por intervalo; verificar links pessoais e diferenças de gráfico |
| Retrospectiva | Ano/mês, vazio, virada, duração visível/oculta e fuso | Fixar data/período e offsets; comparar valores e estados; testes históricos de DST continuam separados |
| Administração | Visitante/login/erro, perfil, avatar, conta, chaves ocultas, relay, cinco formatos, export, edição e imagem | Sessão fictícia, confirmação de destino e retorno; estados no-JS quando suportados |
| Confirmações destrutivas | Excluir escuta/perfil, merge/delete global, cancelar, plano obsoleto e repetição | Cópias próprias; conferir escopo, impacto e dados sobreviventes, sem comparar apenas imagem |
| Acessibilidade | Tab/Shift+Tab, foco, escape quando aplicável, rótulos, mensagens, zoom, contraste e alternativa sem JS | Navegação manual e verificações automatizadas complementares; listar falhas e exceções |
| Compatibilidade | URLs canônicas/legadas, assets, API, `/theme-helper`, configuração e retorno 404 | Comparar `GET`/`HEAD` e POST relevante conforme manifesto; não presumir um status único para todas as rotas |

Capturas pareadas em desktop 1440×1000 e celular 390×844; acrescentar 320/360 px,
zoom e viewport intermediário nos casos de risco; verificar também os limites
639/640/641, 767/768, 959/960 e 1023/1024 quando aplicáveis. Yuu, Pearl e Black recebem cobertura
pareada prioritária em páginas representativas; nos demais temas, pelo menos amostra documentada de legibilidade,
cores e contraste, ampliando conforme falhas. Em vez de multiplicar todas as
combinações, registrar matriz caso/tema/viewport e o que ficou sem executar.
Estados de hover/foco e trechos abaixo da dobra precisam de captura própria quando
necessário. Definir limiares e exceções de contraste conforme o contexto; não
prometer conformidade formal sem auditoria apropriada.

## Protocolo de reprodução e evidência

1. Construir React e binário Go a partir da mesma árvore; registrar revisão/estado
   de trabalho, versões, comando, navegador e resultado de build. Criar fixture
   com perfis e credenciais fictícios, fuso, instantes e contagens explícitos.
2. Criar bases equivalentes por cópia/backup SQLite antes de ligar os servidores;
   uma cópia independente para cada jornada que altera dados. Desativar serviços
   musicais externos, limitar o navegador a loopback e controlar fontes externas.
   Reproduzir fontes obtidas localmente quando disponíveis ou documentar fallback;
   não inventar paridade tipográfica quando o carregamento falhar.
3. Iniciar servidores isolados em portas locais e aguardar hidratação React,
   fontes/imagens e rede relevante; fixar viewport, escala, tema, timezone, idioma,
   movimento e instante de referência quando possível. Quando não for possível
   controlar o relógio, usar período absoluto e registrar a limitação.
4. Exercitar a mesma intenção, não necessariamente a mesma disposição: React pode
   usar modal e Go página; registrar rota real, ação, perfil, estado, região,
   resultado esperado/observado, permissões, JS/no-JS e diferenças aceitáveis.
5. Salvar pares e, quando útil, comparação lado a lado em `comparisons/roadmap-06/`;
   manifesto com data, commit/estado, fixture, modo, navegador, viewport/escala,
   tema, fuso, instante, rota/query sanitizada, estado, rolagem, nome de arquivo e
   resultado. Nunca capturar chave, senha, token, cookie ou informações pessoais.
6. Após ações verificar contagens por perfil, IDs, `integrity_check` e
   `foreign_key_check`; encerrar processos. Erros, casos não executados e
   diferenças aprovadas devem continuar explícitos, com referência à evidência.

`scripts/validate_webui.py` e `validate_admin_webui.py` criam bancos novos;
`validate_multiuser.py` inclui v2→v3. Preparar testes adicionais de migração para
as jornadas web/admin, sem editar manualmente a versão do esquema. A fixture 016
fixa timestamps, mas não controla o relógio do servidor. `capture_admin_webui.cjs`
já ajuda nas capturas e usou portas locais; revisar sua dependência de Playwright,
fontes remotas e seletores antes de reutilizar, sem tratar screenshots passadas
como execução nova. Referências Google Fonts e consulta de releases da SPA devem
ser bloqueadas ou simuladas nos testes e registradas.

## Sequência de issues locais

1. [022 — matriz, fixture e protocolo visual](<../../tasks/iss-022 - 022-—-Preparar-matriz-fixture-e-protocolo-de-comparação.md>).
2. [023 — páginas públicas, mídias e retrospectiva](<../../tasks/iss-023 - 023-—-Comparar-páginas-públicas-mídia-e-retrospectiva.md>).
3. [024 — administração, acessibilidade e ações](<../../tasks/iss-024 - 024-—-Comparar-administração-ações-e-acessibilidade.md>).
4. [025 — migração, compatibilidade e dependências](<../../tasks/iss-025 - 025-—-Validar-migração-compatibilidade-e-dependências-React.md>).
5. [026 — integração, revisão e decisão](<../../tasks/iss-026 - 026-—-Integrar-evidências-e-preparar-decisão-visual.md>).

As issues são locais; as issues 022–025 foram resolvidas e a 026 foi aceita; a correção do helper está registrada na 028. Resolver defeitos reproduzidos com regressão;
não antecipar uma mudança de comportamento sem registrar o caso. As issues 023 e
024 dependem da fixture/protocolo da 022; a 025 pode usar a mesma base; a 026
consolida resultados e parecer. Capturas são prova visual, não substituem testes
de autorização, isolamento ou persistência.

## Revisão Advisor

Sol, esforço alto: **aprovado com ressalvas**. Pela leitura, seleção de perfil e
busca na sidebar Go usam dialogs acionados por JavaScript: ainda não foi
reproduzido se há caminho acessível completo sem JS. A 023 deve verificar isso
no navegador. O capturador antigo espera no máximo um segundo pelas fontes e
150 ms antes das imagens, sem garantir estado estabilizado; a 022 exige prontidão
observável. O Advisor confirmou que `/theme-helper` e `/favicon.svg` mantêm
dependências React. Não houve testes ou capturas nesta revisão.

## Porta de saída e decisão futura

Para propor aceite: jornadas e casos críticos executados com os dois modos; sem
falha crítica aberta de dados, autorização ou integridade; diferenças visuais e
funcionais classificadas com evidência; limitações de 018–021 executadas ou
mantidas como pendências com responsável/decisão; migração v2→v3 exercitada; testes
Go, cliente e validadores relevantes aprovados; acessibilidade e no-JS relatados
com alcance preciso; Advisor revisa resultado. O usuário decide o aceite e as
ressalvas. Ausência de execução não é aprovação.

A promoção de Go a padrão foi solicitada pelo usuário e executada na 027.
Qualquer remoção de React exige
proposta específica, inventário de `/theme-helper`, assets, rotas, configuração,
instalação/atualização, builds, CI, documentos, rollback e testes, seguida de
**autorização explícita separada**. Até essa decisão, React permanece disponível somente por seleção explícita.
Não prometer binário sem dependências nativas ou ganhos de desempenho sem medição:
a API atual usa CGO/libvips e o processo de release ainda inclui o frontend.

## Atualização — issue 022

Fechada e resolvida em 05/11/2026. Fixture v1, matriz, protocolo e manifesto
foram preparados e verificados em
servidores e SQLite descartáveis. [Resultados e limites](../../evidencias/roadmap-06/ISSUE_022.md).
As comparações visuais e jornadas completas continuam nas issues 023–026.

## Atualização — issue 023

Fechada e resolvida em 05/11/2026, com limitações registradas. Jornadas públicas
e correção da navegação sem JavaScript implementadas;
[comparação, evidências e limites](../../evidencias/roadmap-06/ISSUE_023.md).
A validação de fontes carregadas, acessibilidade ampla e o aceite das diferenças
continuam para as próximas issues. Posteriormente, a 027 alterou o padrão para Go.

## Atualização — issue 024

Fechada e resolvida em 05/11/2026, com limitações registradas. Administração e
acessibilidade foram avaliadas com correções de contraste e
nomes acessíveis. [Resultados, comparações e limites](../../evidencias/roadmap-06/ISSUE_024.md).
A avaliação visual final, fontes e auditoria mais ampla permanecem para a 026.

## Atualização — issue 027

A pedido do usuário, Go passou a ser o padrão e a SPA não é fallback no modo Go.
O F5 do VS Code inicia Go diretamente; React continua disponível por seleção explícita.
Os aceites visuais e a decisão sobre remover React permanecem pendentes.

## Atualização — issue 025

Quatro bases v2 fictícias foram migradas no startup; backup, restauração em
cópia, caso com WAL, jornadas e rotas foram verificados. [Resultados e limites](../../evidencias/roadmap-06/ISSUE_025.md).
A falha React do `/theme-helper` foi reproduzida e registrada na
[issue 028](<../../tasks/iss-028 - 028-—-Corrigir-falha-de-contexto-no-theme-helper-React.md>), posteriormente resolvida.

## Atualização — issue 026

Matriz, regressões e ressalvas consolidadas no [relatório de integração](../../evidencias/roadmap-06/ISSUE_026.md).
O usuário aceitou o resultado e suas ressalvas em 05/11/2026; a retirada React foi autorizada em seguida.

## Atualização — issue 028

O helper React usa prévia fictícia e aplica/restaura temas sem depender de perfil.
[Testes e limites](../../evidencias/roadmap-06/ISSUE_028.md); a 026 foi posteriormente aceita.

## Atualização — issue 030

O usuário autorizou a retirada do React, concluída na [issue 030](<../../tasks/iss-030 - 030-—-Retirar-React-e-consolidar-interface-Go.md>). Os trechos anteriores preservam o contexto histórico.
