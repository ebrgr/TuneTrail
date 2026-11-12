---
id: doc-016
title: Estudo inicial das tarefas pendentes
type: other
created_date: '2026-11-12T23:39:00Z'
updated_date: '2026-11-12T23:39:00Z'
tags:
  - estudo
---
> Importado de [docs-dev/ESTUDO_TAREFAS_PENDENTES.md](../../../docs-dev/ESTUDO_TAREFAS_PENDENTES.md).

# Estudo inicial das tarefas pendentes

Data: 06/11/2026. Fonte: [roadmap atualizado](../../../docs-dev/ROADMAP.md), lido após o usuário
salvar sua nova versão. Este documento é um estudo inicial, não autoriza nem
antecipa mudanças de comportamento. As issues locais 001–030 estão fechadas;
a 026 foi aceita e a 030 retirou React. Trechos da sequência inicial no roadmap
que descrevem React opcional, 026 planejada e 028 aberta são históricos e não
correspondem mais à aplicação. Não editar o roadmap recém-salvo para preservá-lo.

O esquema atual é SQLite **v3 consolidado**, conforme [BANCO_DE_DADOS](../../../docs-dev/BANCO_DE_DADOS.md).
Mudanças futuras de esquema devem partir de v3, com nova migração e testes de
upgrade/backup; não alterar migrações já aplicadas. Continuar usando HTML Go,
CSS e apenas o JavaScript necessário, com serviços compartilhados pela API e
pelas páginas, preservando a ingestão ListenBrainz.

## Visão geral na ordem do usuário

| Item | Estado observado | Estudo / dependências antes de implementar |
| --- | --- | --- |
| 1. Retirar CGO/libvips | `bimg` é usado no cache e validação de imagens; build da API com `CGO_ENABLED=0` falha. SQLite usa `modernc.org/sqlite`. | [Estudo técnico inicial](../../../docs-dev/ESTUDO_01_GO_SEM_CGO.md): formatos, WebP, cache existente, limites, desempenho e distribuição. Sem escolha de biblioteca. |
| 2. Administração e privacidade | Layout Go administrativo existe; login e ações são de administrador. Perfis públicos não têm proprietário/visibilidade no modelo atual. | Separar identidade, perfil e chave de ingestão; política central para HTML, API e serviços, migração e matriz de permissões. Badge administrativo é subentrega visual independente, baseada na sessão validada. |
| 3. Nome e identidade | Roadmap propõe TuneTrail, MuseTrail e Songtrail. `assets/icon.png` não estava presente no diretório consultado em 06/11/2026. | Confirmar nome/arquivo, autoria, licença e uso. Avaliar conflitos de marca, nomes e domínios antes de escolher; não afirmar disponibilidade. Planejar nomes de binário, imagens, docs, env, API e créditos ao upstream. |
| 4. TOML | Configuração vem de variáveis em `internal/cfg`; `cmd/api` oferece sufixo `_FILE`; `reset-password` usa o mesmo leitor. | Inventariar chaves e segredos, definir precedência TOML/ambiente/`_FILE`/padrões, localização, permissões, erros e migração compatível; validar execução, Docker e reset de senha. |
| 5. Multilíngue | Templates Go possuem textos e `lang="en"` fixos; datas e JS também apresentam texto. | Definir idiomas, fallback, preferência por usuário/navegador, pluralização, datas, acessibilidade e escape; catalogar textos públicos/admin e erros. Não traduzir nomes reais de artistas. |
| 6. Serviços musicais | Já há importação de arquivos Spotify, API ListenBrainz e relay; imagens Deezer são metadados, não integração de escutas. | Distinguir importação de arquivo, sincronização recorrente e scrobble. Validar APIs e condições de cada serviço, autorização, revogação, cursor, limites, deduplicação, falhas e segredos por perfil. |
| 7. Imagens para redes | `internal/summary` calcula retrospectiva; há código antigo de imagem, mas `GenerateImage` está comentado e não é funcionalidade pronta. | Definir dimensão, formato, layout, fonte, licença de capas, idioma, visibilidade e geração/download sem divulgar perfis privados. Reaproveitar dados da retrospectiva, não consultar a própria API por HTTP. |

## Dependências e cuidados

- **Privacidade:** o `ProfileScope` da API e os handlers HTML resolvem dados em
  caminhos distintos. Uma política compartilhada deve cobrir seleção, home,
  busca, rankings, histórico, mídia, retrospectiva, now-playing, imagens,
  rotas legadas, cache e exportação. Testar visitante, dono, outra pessoa,
  administrador e chave de ingestão; não presumir que ocultar a tela protege a API.
- **Contas:** a sessão e ações atuais exigem admin. Definir relação conta-perfil,
  recuperação e alteração de senha, sessões, transferência/exclusão, e escopo
  de apagar **somente as próprias escutas**, sem conceder alterações no catálogo
  global. Não interpretar “senha simples” como autorização para senha insegura.
  O Advisor encontrou `KOITO_LOGIN_GATE` sem consumidor de produção por busca
  estática; isso não comprova proteção para perfis privados.
- **Badge:** definir se identifica administrador autenticado na sessão atual ou
  associação da pessoa ao perfil exibido. Não marcar automaticamente todo perfil
  apenas porque um administrador abriu a página; validar o ator no servidor.
- **TOML e credenciais:** preferência proposta a discutir: padrões < TOML <
  ambiente explícito, com regra clara para `_FILE`, sem registrar valores
  secretos em logs. A precedência é proposta, não decisão tomada. Rever também
  falhas em `cfg.Load` e configuração Subsonic apontadas pelo Advisor.
- **Integrações:** o relay atual é best-effort, não uma fila persistente. O
  endpoint oficial [Spotify recently played](https://developer.spotify.com/documentation/web-api/reference/get-recently-played)
  exige autorização e informa máximo de 50 itens por resposta; isso não prova
  histórico completo nem disponibilidade igual para todas as contas. Deezer e
  Tidal ainda exigem verificação de documentação, acesso e políticas aplicáveis.
  Não atribuir escuta a um perfil sem vínculo explícito.
- **Ressalvas existentes:** manter visíveis os limites registrados em
  [021](../../../docs-dev/comparisons/roadmap-05/ISSUE_021.md) e [026](../../../docs-dev/comparisons/roadmap-06/ISSUE_026.md):
  datas civis excepcionais, falhas de disco/interleavings, acessibilidade e fontes.
  As issues fechadas não equivalem a testes exaustivos.

## Proposta de sequência de execução, sujeita à decisão do usuário

1. Fazer prova controlada da tarefa 1 e decidir o formato/codec sem alterar o
   armazenamento. Registrar uma issue local com critérios antes do código.
2. Estudar separadamente o badge e o modelo de identidade/privacidade; definir
   autorização e migração antes de permitir autogestão ou sincronização por conta.
3. Estudar o TOML e a internacionalização antes de acrescentar muitas novas telas;
   a identidade visual pode ser estudada em paralelo, após localizar o ícone.
4. Definir arquitetura e testar um serviço de música por vez, depois planejar
   as imagens sociais com as regras de privacidade e identidade já definidas.

A ordem acima é sugestão técnica: não altera a prioridade registrada pelo usuário.
Nenhuma integração, nome, mudança de banco ou dependência foi escolhida neste estudo.

## Revisão Advisor

[Advisor](../../../docs-dev/agents/advisor.md), Sol esforço médio: **reprovado com ressalvas**
para implementar diretamente o roadmap. Por leitura, apontou riscos de
autorização compartilhada HTML/API, sessões apenas admin, imagens públicas,
relay best-effort e configuração; não executou testes. Recomenda separar o
estudo de CGO da migração de contas e definir políticas antes de mudanças.
