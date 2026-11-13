---
id: doc-017
title: Revisão das issues e pendências remanescentes
type: other
created_date: '2026-11-12T23:39:00Z'
updated_date: '2026-11-12T23:40:00Z'
tags:
  - estudo
---
> Importado de `docs-dev/REVISAO_ISSUES_PENDENCIAS.md`. Original removido em 12/11/2026; consulte o histórico do Git.

# Revisão das issues e pendências remanescentes

Data: 09/11/2026. Escopo: 47 issues Markdown em `docs-dev/issues`, índice,
roadmap, relatórios de aceite e conferência estática dirigida do código atual.
A lista remota do GitHub não foi confirmada: a revisão automática de permissão
para a consulta expirou; a alternativa sem elevação falhou por DNS. Os números
abaixo são **locais**, não contagens de issues publicadas no GitHub.
Nenhuma issue foi encerrada, reaberta ou publicada nesta revisão.

## Atualização após a decisão do usuário — 09/11/2026

Após a auditoria abaixo, o usuário determinou o fechamento das issues 043–045,
renomeadas com `-solved.md`, e a abertura da
[048 — documentação e site TuneTrail](<../../tasks/issue-048 - 048-—-Documentação-e-site-oficial-do-TuneTrail.md>).
Naquela decisão havia **48 issues locais: 42 fechadas e 6 abertas** (038–042 e 048). Posteriormente, as 039–041 foram fechadas e a 042 implementada; ela aguarda aceite. Posteriormente, foi aberta a issue 049 para associação residual entre artista e álbum. Posteriormente, foram planejadas as issues 050–054 para a interface multilíngue. A 050 foi fechada por aprovação do usuário. A 051 também foi fechada por aprovação do usuário. Com a abertura da [055 — metadados musicais](<../../tasks/issue-055 - 055-—-Aproveitar-tags-e-datas-de-lançamento-das-músicas.md>) e da [056 — interações da UI](<../../tasks/issue-056 - 056-—-Estudar-interações-mais-dinâmicas-na-interface.md>), e da [057 — controles da retrospectiva](<../../tasks/issue-057 - 057-—-Ajustar-controles-da-retrospectiva.md>), o estado atual é 47 fechadas e 10 abertas.
As pendências de documentação/site passam à 048; as limitações de verificação
continuam registradas. O inventário e as recomendações abaixo descrevem o momento
anterior a essa decisão e não representam o status atual de 043–045.

## Inventário das abertas na auditoria original

O índice e os arquivos registram **8 abertas e 39 fechadas**.

| Issues | Estado encontrado | Encaminhamento sugerido |
| --- | --- | --- |
| [038](<../../tasks/issue-038 - 038-—-Contrato-e-decisões-da-configuração.md>) | Contrato TOML não implementado; decisões prévias ainda necessárias. | Manter em espera conforme decisão do usuário. Resolve precedência, segredos, bootstrap e recuperação antes de 039–042. |
| [039](<../../tasks/issue-039 - 039-—-Leitura-e-validação-de-conf.toml.md>) | Leitura tipada e validação ainda planejadas. O carregador atual continua baseado no ambiente. | Depende da 038. |
| [040](<../../tasks/issue-040 - 040-—-Persistência-e-concorrência-da-configuração.md>) | Escrita segura, conflitos de revisão e persistência ainda planejados. | Depende de 038–039. |
| [041](<../../tasks/issue-041 - 041-—-Edição-de-configuração-pelo-master.md>) | Interface master para configuração ainda planejada. | Depende de 038–040; preservar sessão, origem, CSRF e proteção dos segredos. |
| [042](<../../tasks/issue-042 - 042-—-Integração-e-aceite-da-configuração.md>) | Integração/aceite ainda não executados. | Depende de 038–041. |
| [043](<../../tasks/issue-043 - 043-—-Identidade-visual-TuneTrail.md>) | Identidade aplicada; SVG/PNG aceitos. O estudo registra ausência de comparação visual após a mudança e build da documentação sem verificação. | Fazer aceite desktop/celular da identidade atual, ícones e rodapé; validar documentação antes de propor fechamento. |
| [044](<../../tasks/issue-044 - 044-—-Versionamento-próprio-do-TuneTrail.md>) | Implementação e correções de versionamento presentes; status ainda aberto. | Candidata a revisão de encerramento. Conferência desta auditoria passou; não representa nova execução da suíte Go completa nem publicação. |
| [045](<../../tasks/issue-045 - 045-—-Links-e-diretório-do-TuneTrail.md>) | Links, módulo/destino da imagem e mudança local executados. A configuração Astro ainda usa `site: https://koito.io`. | Separar entrega concluída da pendência do endereço da documentação; não inventar domínio. Publicação/acessibilidade remota não confirmadas nesta auditoria. |

## Melhorias registradas nas fechadas

Prioridade abaixo é proposta técnica, não autorização para implementar.
As limitações aceitas não significam necessariamente defeito reproduzido atual.

| Origem | Ponto que continua relevante | Proposta de trabalho e prioridade |
| --- | --- | --- |
| [046](<../../tasks/issue-046 - 046-—-Migração-segura-de-koito.db-para-tunetrail.db.md>) | Não houve injeção sistemática de erro de disco/interrupção em cada fase, nem restauração operacional completa. O protocolo exige filesystem local com lock/hard link e processo antigo parado; a validação confere presença, não equivalência integral dos triggers. | **Alta:** testar recuperação de migração interrompida e restauração em diretórios descartáveis, incluindo WAL, conflitos, falta de espaço e falha de publicação. Avaliar triggers separadamente. Suporte a volumes de rede exige decisão própria; não é suporte atual prometido. |
| [020/021](../../evidencias/roadmap-05/ISSUE_020.md) | Reinício no meio de importação, falhas de disco e certas ordens de operações concorrentes não foram integralmente exercitados. | **Alta:** testes dirigidos de recuperação, falhas e preservação das escutas já gravadas. Importação parcial e exportação dinâmica são contratos atuais; torná-las atômica/snapshot seria funcionalidade nova. |
| [024/026](../../evidencias/roadmap-06/ISSUE_026.md) | Leitor de tela completo, zoom real de 200%, contraste de todos os estados, nove temas adicionais e fontes efetivamente carregadas ficaram sem validação abrangente. | **Média:** auditoria da interface Go atual, já com TuneTrail. Incluir foco, teclado, erros de formulário, tabelas, modal, temas e larguras pequenas; não repetir a matriz React removida. Pode compartilhar capturas com o aceite da 043. |
| [018](../../evidencias/roadmap-05/ISSUE_018.md) | Contagens são filtradas por período, mas dias ativos/sequência são históricos; a média combina as duas bases. Escutas futuras dentro do dia atual podem participar dos agregados. | **Média:** decidir e documentar a semântica desejada das estatísticas antes de alterar cálculos, HTML ou API. É compatibilidade preservada, não correção pendente já aprovada. |
| [018](../../evidencias/roadmap-05/ISSUE_018.md) | Dia civil inteiramente removido, como Apia 30/12/2011, pode ser pulado/duplicado no calendário. Entrada manual não suporta offsets históricos com segundos. | **Baixa**, salvo demanda real: especificar representação e testes desses calendários; não anunciar suporte universal a fusos. |
| [018/020](../../evidencias/roadmap-05/ISSUE_018.md) | Agregação SQL pode processar um grupo por segundo distinto; não houve benchmark de bases grandes nessa etapa. | **Média:** medir histórico grande e consultas de intervalo antes de propor índices ou reescrita. Não há lentidão atual comprovada nesta auditoria. |
| [020/034](../../evidencias/transicao-imagens/ISSUE_034.md) | Relay permanece best-effort; 429 não é repetido, não há fila durável nem drenagem geral no encerramento. Importação/exportação preservam seus limites documentados. | **Média**, se a meta for entrega confiável: planejar fila persistente, retry/backoff e shutdown antes de ampliar integrações. Não confundir escuta salva localmente com confirmação remota. |
| [034](../../evidencias/transicao-imagens/ISSUE_034.md) | Dockerfile não define USER; o serviço continua executando como root. | **Média:** planejar execução sem root e compatibilidade de permissões/ownership de volumes existentes, incluindo Unraid. Não alterar usuário do container sem migração e teste. |
| [035](<../../tasks/issue-035 - 035-—-Refatorar-o-código-de-imagens-da-retrospectiva-com-imaging.md>) | Auxiliares de imagem foram preparados, mas o gerador social continua desativado e não há rota/botão de exportação. | **Funcionalidade nova:** especificar layout, formatos, fontes, licença, limites e privacidade; depois implementar imagem de retrospectiva para redes sociais. |
| [047](<../../tasks/issue-047 - 047-—-Corrigir-curva-do-gráfico-de-escutas.md>) | Recomendação explícita: curvas mais redondas. A implementação atual usa tangentes horizontais em cada ponto. | **Melhoria visual pequena:** estudar outra interpolação que preserve limites e valores; comparar picos/zeros e manter testes contra overshoot. A correção dos trechos quebrados já está concluída. |
| [037](../../evidencias/transicao-imagens/ISSUE_037.md) | Reservas de publicação e coleta existem apenas no processo. | **Condicional:** coordenar processos somente se for adotado suporte a múltiplas instâncias com cache compartilhado. Não tratar como falha da implantação de uma única instância. |
| [032/036](../../evidencias/transicao-imagens/ISSUE_036.md) | Reader bloqueado e codec iniciado não podem ser interrompidos à força; cotas de admissão não são teto único de toda a memória de rede. | **Condicional:** manter limite explícito e medir carga real. Isolamento de processamento/teto global seria trabalho próprio; não prometer cancelamento instantâneo. |

## Pontos antigos já superados

- As pendências de integração descritas no andamento da 013 são explicitamente
  históricas: foram encaminhadas à 015 e às validações seguintes.
- A falta de comparação numérica por intervalo da 021 foi coberta na 023:
  192 intervalos API e 96 tabelas Go com oráculo independente. Não abrir outra
  tarefa genérica dizendo que isso nunca foi testado. A 025 preserva ressalva
  sobre repetir toda a matriz em banco migrado.
- Jornadas sobre bases migradas avançaram nas 025 e 030. Não repetir a lacuna
  ampla da 021 como se nenhuma jornada posterior tivesse ocorrido; permanece
  restauração com binário antigo não exercitada e cobertura parcial da matriz.
- Falha do theme-helper foi corrigida pela 028; React e suas dependências de
  execução foram retirados pela 030. Rever cliques/hidratação do React não é
  trabalho atual do produto.
- Riscos de admissão/cancelamento da revisão 032/033 foram tratados na 036;
  a corrida de publicação automática versus coleta foi tratada na 037.
- Libvips/bimg não são dependências do aplicativo atual. Os registros do estudo
  inicial que dizem o contrário descrevem um estado anterior.

## Organização documental a corrigir em uma próxima tarefa

1. O roadmap ainda apresenta modo Go opcional, 026 planejada e React disponível.
   O relatório da 030 e `cfg.loadConfig` confirmam Go exclusivo.
2. O estudo inicial de tarefas pendentes contém fotografia anterior à migração
   de imagens e à identidade; marcá-la claramente como histórica ou acrescentar
   referência ao estado atual, preservando as evidências originais.
3. A 046 tem checkboxes de aceite todos desmarcados e o parecer inicial ainda
   diz implementação pendente, embora a seção final registre entrega aprovada.
   Atualizar a apresentação por evidência: marcar somente o que foi exercitado e
   deixar falhas sistemáticas/restauração completa como lacunas reais.
4. Reconciliar 043–045 com entrega e validação atuais, sem fechá-las só porque
   há código nem deixar implementação concluída indistinguível de planejamento.
5. Registrar melhorias escolhidas como novas issues locais com escopo reduzido,
   mantendo o aceite das issues anteriores e os contratos preservados.

## Sequência sugerida

1. Reconciliar os registros; validar 043 e avaliar o encerramento da 044.
2. Resolver a pendência da documentação na 045 quando houver endereço definido.
3. Priorizar testes de recuperação de banco/importação e acessibilidade da UI Go.
4. Como melhoria visual pequena, estudar a recomendação da 047.
5. Retomar TOML 038–042 somente quando o usuário retirar a etapa da espera.
   Imagens sociais, relay durável e múltiplas instâncias precisam de planejamento
   próprio; não são retomados automaticamente por esta auditoria.

## Verificações desta revisão

- Conferência dos 47 nomes/status e busca de pendências nos textos e relatórios.
- Consultas ao Graphify seguidas de leitura dirigida de configuração, curva,
  bootstrap relacionado, relay, versionamento, Dockerfile e configuração Astro.
- `python3 scripts/version.py`: TuneTrail 0.6 sincronizado.
- `python3 -m unittest discover -s scripts -p test_version.py`: quatro testes passaram.
- Links relativos nas 47 issues, índice e roadmap: nenhum destino ausente encontrado.
- Sem execução de suíte Go completa, falhas simuladas novas, inspeção visual nova
  ou verificação de publicação remota. Os resultados anteriores citados continuam
  sendo evidências históricas dos respectivos relatórios.
