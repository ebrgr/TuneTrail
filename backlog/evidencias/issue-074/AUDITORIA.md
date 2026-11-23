# ISSUE-074 — Auditoria responsiva da administração

Chromium em loopback, com servidores isolados e bancos SQLite fictícios em diretório temporário fora da worktree. Cada servidor definiu seu tema padrão para que Yuu e Pearl funcionassem também com JavaScript desligado. A matriz inicial cobriu 17 páginas ou estados e o download de exportação, em 390, 768 e 1440 px, nos dois temas e com JavaScript ligado e desligado. Foram 216 verificações de página ou download. Os tamanhos de viewport e os temas efetivos foram medidos no navegador.

A matriz inicial mediu pelo menos 44 px de altura nos controles principais e verificou a largura do documento. As capturas de página inteira foram feitas com JavaScript ligado. A exportação é um download e está representada pelo cartão no Histórico.

## Resultado por página

| Página | Resultado |
| --- | --- |
| Conta | Identidade, senha e sessão em cartões. Zona de perigo em largura total. Estados normal, hover e pressed verificados nos botões primário e secundário. |
| Perfis e contas | Lista e criação em cartões. A lista passa a duas colunas no desktop. |
| Perfil | Identidade e avatar em uma coluna a 390 e 768 px e duas a 1440 px. |
| Gestão de conta | Acesso, papel e zona de perigo separada. |
| Chaves de API | O detalhe tinha 21–24 px. Passou a ter pelo menos 44 px. A revogação confirmada ficou num quadro separado. Renomear persistiu com POST nativo (303) e com troca do cartão (200, fragmento profile-keys). |
| Relay | URL e token têm rótulos acima dos campos. Salvar, testar e remover ficam no cartão. Captura com configuração vazia. |
| Importação e exportação | Cartões lado a lado a 1440 px e empilhados a 390 e 768 px. Exportar pelo link gerou o arquivo tunetrail_export.json. As jornadas de importação e exportação também passaram em make check. |
| Configuração do servidor | Cartões por área, detalhes expansíveis e ações alcançáveis. A geração de parâmetros continua disponível com JavaScript. Captura com credenciais vazias. |
| Saúde da ingestão | Atalhos de 36 px passaram a 44 px. Cartões ganharam borda fina e raio padrão. Inbox precede Outbox e as colunas ficam lado a lado a 1440 px. Descarte em cartão próprio. |
| Revisão de catálogo | A revisão de mesclagem da faixa mostrou resumo, mudanças e zona de perigo, com leitura em uma coluna. |
| Revisão de exclusão | Resumo e efeitos separados da confirmação de exclusão da conta e do perfil. |
| Login | Formulário central com rótulos acima dos campos. Login de administrador e usuário comum exercitado na matriz. |
| Resgate | Formulário central com instruções, rótulos e campos vazios. |
| Saída | Página de confirmação e POST de saída exercitados. A página protegida retornou 401 após encerrar a sessão. |
| Erro 401 | Cartão de login após acesso anônimo à confirmação de saída. |
| Erro 403 | Cartão de acesso negado na configuração do servidor para usuário comum. |
| Erro 404 | Cartão de página indisponível ao revisar perfil inexistente. |

O teste de edição de chave encontrou um POST incorreto para a URL da página quando o botão não declarava formaction. O aprimoramento agora lê o atributo do botão e usa a action do formulário quando ele está ausente. O teste confirmou o destino, o cabeçalho do fragmento e a persistência do rótulo após recarregar.

## Revisão após o retorno do usuário

Removi o teto de 73,75 rem de `.admin-shell`. Uma única regra de estrutura vale para a administração. A área usa a largura do viewport descontando as margens internas e, a partir de 40 rem, a barra lateral. Login, resgate, saída e erros mantêm o cartão de leitura centralizado. A partir de 64 rem, um cartão sem par ocupa a linha inteira. Conta, Perfil, Configuração do servidor e Histórico mantêm duas colunas onde há pares. No Histórico, Importar e Exportar dividem a linha.

A chave recém-criada foi aberta, lida, copiada e revogada em um banco fictício. O valor usa bloco monoespaçado com quebra segura e o botão Copiar fica abaixo. Com JavaScript desligado, é possível selecionar a chave no próprio bloco. No teste de Clipboard indisponível, o botão selecionou o valor para cópia manual. As capturas de Chaves mostram o detalhe aberto com caracteres fictícios do mesmo comprimento. Senhas, chaves reais e tokens não aparecem.

O script de estados aplica `:hover` e `:active` pelo Chrome DevTools Protocol e compara a cor de fundo computada entre os três estados de cada botão, detalhe expansível e link visível da área administrativa. O escopo inclui as exceções do relay por destino, a confirmação da exceção e as revisões de catálogo e exclusão. A checagem estática do repositório agora exige os estados no seletor compartilhado.

As larguras abaixo são medidas no Chromium e arredondadas ao pixel. Yuu e Pearl deram as mesmas medidas. Em 1440 px, Chaves e Relay passaram de 542 px para 1302 px.

| Área ou cartão | 390 px | 768 px | 1440 px |
| --- | ---: | ---: | ---: |
| Área útil (.admin-main) | 358 | 630 | 1302 |
| Conta | 358 / 358 / 358 / 358 | 630 / 630 / 630 / 630 | 643 / 643 / 1302 / 1302 |
| Perfil | 358 / 358 | 630 / 630 | 643 / 643 |
| Chaves de API | 358 | 630 | 1302 |
| Relay | 358 | 630 | 1302 |
| Histórico | 358 / 358 | 630 / 630 | 643 / 643 |
| Configuração do servidor | 6 × 358 | 6 × 630 | 6 × 643 |
| Saúde da ingestão | 358 / 358 | 630 / 630 | 639 / 639 |

Cada valor representa um cartão ou, na Saúde, uma coluna de cartões. Em Conta, Sessão e Zona de perigo ocupam a linha inteira. Em Perfis e contas, o último cartão sem par ocupou 1270 px, a largura útil da lista interna.

A matriz passou nas 12 combinações de tema, largura e JavaScript. Foram 294 verificações de páginas ou estados e 1920 instâncias de controles visíveis verificadas nos três estados. O documento coube em todas as larguras. O diálogo Subsonic foi testado nas seis combinações com JavaScript.

## Capturas finais

Foram regenerados os seis cenários de cada uma das sete páginas solicitadas, num total de 42 capturas. As demais imagens são da primeira rodada.

| Página | 390 Yuu | 390 Pearl | 768 Yuu | 768 Pearl | 1440 Yuu | 1440 Pearl |
| --- | --- | --- | --- | --- | --- | --- |
| Conta | [Yuu](capturas/account-390-yuu.png) | [Pearl](capturas/account-390-pearl.png) | [Yuu](capturas/account-768-yuu.png) | [Pearl](capturas/account-768-pearl.png) | [Yuu](capturas/account-1440-yuu.png) | [Pearl](capturas/account-1440-pearl.png) |
| Perfis e contas | [Yuu](capturas/profiles-390-yuu.png) | [Pearl](capturas/profiles-390-pearl.png) | [Yuu](capturas/profiles-768-yuu.png) | [Pearl](capturas/profiles-768-pearl.png) | [Yuu](capturas/profiles-1440-yuu.png) | [Pearl](capturas/profiles-1440-pearl.png) |
| Perfil | [Yuu](capturas/profile-390-yuu.png) | [Pearl](capturas/profile-390-pearl.png) | [Yuu](capturas/profile-768-yuu.png) | [Pearl](capturas/profile-768-pearl.png) | [Yuu](capturas/profile-1440-yuu.png) | [Pearl](capturas/profile-1440-pearl.png) |
| Gestão de conta | [Yuu](capturas/accounts-390-yuu.png) | [Pearl](capturas/accounts-390-pearl.png) | [Yuu](capturas/accounts-768-yuu.png) | [Pearl](capturas/accounts-768-pearl.png) | [Yuu](capturas/accounts-1440-yuu.png) | [Pearl](capturas/accounts-1440-pearl.png) |
| Chaves de API | [Yuu](capturas/keys-390-yuu.png) | [Pearl](capturas/keys-390-pearl.png) | [Yuu](capturas/keys-768-yuu.png) | [Pearl](capturas/keys-768-pearl.png) | [Yuu](capturas/keys-1440-yuu.png) | [Pearl](capturas/keys-1440-pearl.png) |
| Relay | [Yuu](capturas/relay-390-yuu.png) | [Pearl](capturas/relay-390-pearl.png) | [Yuu](capturas/relay-768-yuu.png) | [Pearl](capturas/relay-768-pearl.png) | [Yuu](capturas/relay-1440-yuu.png) | [Pearl](capturas/relay-1440-pearl.png) |
| Importação e exportação | [Yuu](capturas/import-390-yuu.png) | [Pearl](capturas/import-390-pearl.png) | [Yuu](capturas/import-768-yuu.png) | [Pearl](capturas/import-768-pearl.png) | [Yuu](capturas/import-1440-yuu.png) | [Pearl](capturas/import-1440-pearl.png) |
| Configuração do servidor | [Yuu](capturas/settings-390-yuu.png) | [Pearl](capturas/settings-390-pearl.png) | [Yuu](capturas/settings-768-yuu.png) | [Pearl](capturas/settings-768-pearl.png) | [Yuu](capturas/settings-1440-yuu.png) | [Pearl](capturas/settings-1440-pearl.png) |
| Saúde da ingestão | [Yuu](capturas/health-390-yuu.png) | [Pearl](capturas/health-390-pearl.png) | [Yuu](capturas/health-768-yuu.png) | [Pearl](capturas/health-768-pearl.png) | [Yuu](capturas/health-1440-yuu.png) | [Pearl](capturas/health-1440-pearl.png) |
| Revisão de catálogo | [Yuu](capturas/catalog-390-yuu.png) | [Pearl](capturas/catalog-390-pearl.png) | [Yuu](capturas/catalog-768-yuu.png) | [Pearl](capturas/catalog-768-pearl.png) | [Yuu](capturas/catalog-1440-yuu.png) | [Pearl](capturas/catalog-1440-pearl.png) |
| Revisão de exclusão | [Yuu](capturas/deletion-390-yuu.png) | [Pearl](capturas/deletion-390-pearl.png) | [Yuu](capturas/deletion-768-yuu.png) | [Pearl](capturas/deletion-768-pearl.png) | [Yuu](capturas/deletion-1440-yuu.png) | [Pearl](capturas/deletion-1440-pearl.png) |
| Login | [Yuu](capturas/login-390-yuu.png) | [Pearl](capturas/login-390-pearl.png) | [Yuu](capturas/login-768-yuu.png) | [Pearl](capturas/login-768-pearl.png) | [Yuu](capturas/login-1440-yuu.png) | [Pearl](capturas/login-1440-pearl.png) |
| Resgate | [Yuu](capturas/redeem-390-yuu.png) | [Pearl](capturas/redeem-390-pearl.png) | [Yuu](capturas/redeem-768-yuu.png) | [Pearl](capturas/redeem-768-pearl.png) | [Yuu](capturas/redeem-1440-yuu.png) | [Pearl](capturas/redeem-1440-pearl.png) |
| Saída | [Yuu](capturas/logout-390-yuu.png) | [Pearl](capturas/logout-390-pearl.png) | [Yuu](capturas/logout-768-yuu.png) | [Pearl](capturas/logout-768-pearl.png) | [Yuu](capturas/logout-1440-yuu.png) | [Pearl](capturas/logout-1440-pearl.png) |
| Erro 401 | [Yuu](capturas/401-390-yuu.png) | [Pearl](capturas/401-390-pearl.png) | [Yuu](capturas/401-768-yuu.png) | [Pearl](capturas/401-768-pearl.png) | [Yuu](capturas/401-1440-yuu.png) | [Pearl](capturas/401-1440-pearl.png) |
| Erro 403 | [Yuu](capturas/403-390-yuu.png) | [Pearl](capturas/403-390-pearl.png) | [Yuu](capturas/403-768-yuu.png) | [Pearl](capturas/403-768-pearl.png) | [Yuu](capturas/403-1440-yuu.png) | [Pearl](capturas/403-1440-pearl.png) |
| Erro 404 | [Yuu](capturas/404-390-yuu.png) | [Pearl](capturas/404-390-pearl.png) | [Yuu](capturas/404-768-yuu.png) | [Pearl](capturas/404-768-pearl.png) | [Yuu](capturas/404-1440-yuu.png) | [Pearl](capturas/404-1440-pearl.png) |

## Verificação de código

- `make check` passou com 13 de 13 jornadas E2E.
- A referência anterior coincidiu com o HEAD antes da alteração. A regeneração alterou quatro arquivos CSS/JS da referência e os identificadores de cache em 24 páginas. Após atualizar a referência versionada, `scripts/compare_ui_reference.py` confirmou igualdade nos 190 arquivos.
- Nesta revisão, a referência foi regenerada para incorporar o CSS administrativo e o JavaScript de cópia. A comparação final coincidiu nos 190 arquivos.
- O aceite visual do conjunto cabe ao usuário.
