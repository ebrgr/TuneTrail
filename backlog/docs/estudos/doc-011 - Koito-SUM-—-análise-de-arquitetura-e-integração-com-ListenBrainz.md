---
id: doc-011
title: Koito-SUM — análise de arquitetura e integração com ListenBrainz
type: other
created_date: '2026-11-12T23:39:00Z'
updated_date: '2026-11-12T23:39:00Z'
tags:
  - estudo
---
> Importado de [docs-dev/Koito_SUM_Analise_1a_análise.md](../../../docs-dev/Koito_SUM_Analise_1a_análise.md).

# Koito-SUM — análise de arquitetura e integração com ListenBrainz

**Consolidação da conversa até 4 de novembro de 2026.**  
**Repositório:** https://github.com/ebrgr/TuneTrail
**Base inicial analisada:** commit a936c1c1b050db486a67c6d429a4c0960d83f790, de 31/10/2026.

Este documento reúne as conclusões, verificações de código e propostas discutidas. Não constitui uma implementação: não foram realizadas a migração da interface nem a sincronização com ListenBrainz. Não houve execução completa da aplicação, teste de desempenho ou consulta à conta pessoal do usuário no ListenBrainz. Alterações posteriores do repositório podem exigir nova conferência.

## 1. Objetivos discutidos

1. Compreender a arquitetura atual do Koito-SUM.
2. Avaliar a substituição do frontend React por templates HTML em Go, mantendo o visual.
3. Verificar a importação de históricos do ListenBrainz.
4. Entender os campos de origem das reproduções e sua persistência no banco.
5. Avaliar sincronização automática ListenBrainz → Koito, sobretudo para escutas do Spotify.
6. Evitar duplicatas e ciclos, pois o Koito também encaminha escutas ao ListenBrainz.

## 2. Arquitetura atual

O projeto separa frontend e backend no código, mas usa uma implantação monolítica.

| Camada | Implementação observada |
| --- | --- |
| Backend | Go, roteador Chi e APIs HTTP |
| Frontend | React 19, TypeScript e React Router |
| Estado e consultas | React Query |
| Estilos | Tailwind CSS, Vanilla Extract e variáveis CSS |
| Gráficos | Recharts e componentes próprios |
| Banco em uso | SQLite |
| Produção | Um servidor Go entrega API e arquivos estáticos |
| Desenvolvimento | Go na porta 4110 e Vite na porta 5173 |

A configuração React Router usa `ssr: false`. A interface funciona como SPA: o servidor entrega o HTML de entrada, o navegador executa React e consulta APIs JSON.

O Dockerfile possui uma etapa Node para compilar a interface e outra para compilar Go. O contêiner final não executa um servidor Node: executa o aplicativo Go e contém os arquivos gerados do frontend.

### Organização funcional

- `cmd/api`: entrada da aplicação.
- `engine`: inicialização, rotas, handlers e middlewares.
- `internal/catalog`: associação de artistas, álbuns, faixas e escutas.
- `internal/db/sqlite`: persistência e consultas.
- `internal/importer`: importadores.
- `internal/summary`: geração das retrospectivas.
- `imagecache`: processamento e cache de imagens.
- `client`: interface React.
- `db/migrations_sqlite`: migrações de banco.

### Fluxo de uma escuta

1. Um cliente envia dados para `/apis/listenbrainz/1/submit-listens`.
2. A chave de ingestão determina o perfil.
3. O backend associa os metadados ao catálogo.
4. A escuta é registrada no SQLite.
5. Havendo relay configurado para o perfil, o conteúdo pode ser encaminhado a outro servidor compatível.
6. A interface consulta estatísticas, rankings, histórico e retrospectivas.

### Perfis e administração

O fork distingue perfis públicos de escuta e sessão administrativa. Não se deve tratar cada perfil como uma conta administrativa independente. Consultas e ingestão precisam preservar o escopo do perfil; ações administrativas exigem autorização.

## 3. Migração para Go e templates HTML

### Viabilidade

É viável substituir a SPA por páginas renderizadas com `html/template`, preservando a identidade visual. React não é requisito para reproduzir cards, tabelas, capas, cores, tipografia, gradientes, barra lateral ou responsividade.

| Elemento atual | Alternativa proposta |
| --- | --- |
| Componentes React | Templates e parciais HTML |
| React Router | Rotas HTTP no Chi |
| React Query | Carregamento dos dados nos handlers Go |
| Tailwind/Vanilla Extract | CSS convencional equivalente |
| Ícones React | SVGs estáticos |
| Recharts | SVG gerado pelo Go para os gráficos existentes |
| Busca e modais dinâmicos | JavaScript pequeno e localizado |
| Atualizações parciais | JavaScript simples ou HTMX, se reduzir complexidade |

Os temas já usam variáveis como `--color-bg` e `--color-primary`, facilitando a migração. A grade de atividade já é feita com elementos HTML e CSS. O gráfico de interesse exige substituir o Recharts e reproduzir sua curva e acabamento.

### Significado de “somente Go”

A proposta é usar Go como linguagem principal da aplicação, eliminando a cadeia Node/TypeScript da interface. O navegador continuará usando HTML e CSS.

Para manter troca instantânea de tema, busca enquanto digita, modais e atualização de “tocando agora”, é recomendável manter JavaScript mínimo. Uma interface sem JavaScript pode usar navegação e formulários tradicionais, mas não terá todos os mesmos comportamentos instantâneos.

HTMX é uma opção, não uma decisão tomada. A recomendação inicial foi Go + html/template + CSS + JavaScript mínimo.

### Benefícios e limites

Benefícios esperados:

- Uma ferramenta principal de compilação.
- Um servidor no desenvolvimento.
- Menos dependências de frontend.
- Menos duplicação de tipos e gerenciamento de estados.
- Possibilidade de incorporar templates e arquivos estáticos com `go:embed`.

Não foi medido ganho de RAM ou velocidade. O contêiner atual já não executa Node, portanto o principal benefício é manutenção e desenvolvimento.

Existe uma dependência adicional: `bimg`/libvips e CGO para imagens. Migrar o frontend não a elimina. Gerar um executável sem bibliotecas nativas seria outro trabalho.

### Estrutura e sequência sugeridas

Preservar banco, migrações, catálogo, importadores e API ListenBrainz. Acrescentar uma camada web com handlers HTML, modelos de apresentação, templates e arquivos estáticos.

Os handlers HTML devem compartilhar serviços e consultas com a API, sem requisitar a própria API via HTTP.

Sequência:

1. Reproduzir layout, fontes, temas, barra lateral e cards.
2. Migrar seleção de perfis, home, rankings e histórico.
3. Migrar artistas, álbuns, faixas e retrospectiva.
4. Migrar administração, importação, chaves, edição e mesclagem.
5. Validar comportamento, permissões, fusos e isolamento entre perfis.
6. Comparar capturas em desktop e celular antes de remover React.

Na base inicialmente analisada, `client/app` tinha aproximadamente 9.145 linhas em 106 arquivos TS/TSX. `AdminProfiles.tsx` tinha 739 linhas. Esses números indicam uma reescrita relevante da apresentação, mesmo com reaproveitamento do backend.

## 4. Importação existente do ListenBrainz

O Koito possui importador específico do ListenBrainz.

No Koito original, a documentação descreve:

1. Solicitar exportação no ListenBrainz.
2. Colocar o ZIP na pasta `import` do diretório de configuração.
3. Garantir que o nome contenha `listenbrainz`.
4. Reiniciar para iniciar a importação.

No Koito-SUM há importação vinculada a perfil, e o Compose recomendado analisado define `KOITO_SKIP_IMPORT=true`. Portanto, não se deve aplicar automaticamente o procedimento de importação na inicialização sem verificar a configuração e o fluxo administrativo da versão instalada. Deve-se definir explicitamente o perfil de destino.

Consultas de enriquecimento ao MusicBrainz podem tornar históricos grandes lentos. Desativar temporariamente o enriquecimento acelera a importação, mas reduz os metadados buscados, como aliases de artistas.

**Distinção:** o importador de arquivos não equivale a uma sincronização on-line recorrente.

No código discutido não foi identificada uma rotina que consulte periodicamente o ListenBrainz e copie novas escutas recebidas de outros serviços. Isso seria uma funcionalidade nova.

## 5. Formato das escutas e campos de origem

A API retorna JSON. A exportação é um ZIP; a documentação consultada descreve histórico em arquivos JSONL, com um objeto JSON por linha. A compatibilidade do importador com a estrutura exata de um arquivo exportado deve ser verificada na implementação.

Exemplo ilustrativo de uma escuta, não uma resposta real da conta do usuário:

```json
{
  "listened_at": 1793530800,
  "track_metadata": {
    "artist_name": "Coldplay",
    "track_name": "Yellow",
    "release_name": "Parachutes",
    "additional_info": {
      "media_player": "Spotify",
      "music_service": "spotify.com",
      "submission_client": "Aplicativo de scrobble",
      "origin_url": "https://open.spotify.com/track/exemplo"
    }
  }
}
```

| Campo | Significado |
| --- | --- |
| listened_at | Horário da reprodução, Unix em segundos |
| artist_name | Artista |
| track_name | Faixa |
| release_name | Álbum, quando informado |
| media_player | Programa usado para ouvir |
| submission_client | Programa ou integração que enviou a escuta |
| original_submission_client | Cliente que originalmente enviou a escuta, quando informado |
| music_service | Domínio canônico do serviço, como spotify.com |
| music_service_name | Nome do serviço quando o domínio não é conhecido |
| origin_url | Endereço de origem da faixa |
| duration_ms ou duration | Duração informada |
| MBIDs | Identificadores MusicBrainz, quando disponíveis |

Os campos adicionais são opcionais. Player, serviço e cliente de envio podem ser diferentes. Por exemplo, BrainzPlayer pode reproduzir conteúdo do Spotify: `media_player` será BrainzPlayer e `music_service` será spotify.com.

Um identificador ou link do Spotify isoladamente pode representar uma correspondência de catálogo. Não deve ser considerado prova suficiente da origem daquela reprodução sem avaliar a semântica do campo e da integração.

## 6. O que o Koito-SUM grava no banco

Foi verificada a coluna `client` na tabela `listens`.

Tanto no recebimento pela API quanto no importador ListenBrainz, a escolha observada é:

1. Usar `media_player`, se preenchido.
2. Caso contrário, usar `submission_client`.
3. Se ambos faltarem, gravar string vazia.

| Metadados recebidos | Valor de client |
| --- | --- |
| media_player = Spotify | Spotify |
| media_player ausente; submission_client preenchido | Nome do cliente |
| Nenhum dos dois | String vazia |

O fluxo analisado não preserva separadamente `music_service`, `origin_url`, player e cliente nessa coluna. Não se deve interpretar `client` como identificação completa e garantida da origem.

Para uma evolução, foi sugerido guardar separadamente player, cliente de envio e serviço de música. Também é útil distinguir **serviço de reprodução** de **caminho de ingestão**: Spotify pode ser o serviço, enquanto ListenBrainz é o sistema de onde o Koito leu o registro. Isso é uma proposta, não o esquema atual.

## 7. Encaminhamento Koito → ListenBrainz

O handler analisado repassa ao relay o corpo original da requisição recebida. A autenticação do destino é configurada pelo Koito.

Consequências:

- Os metadados originais do cliente podem acompanhar o encaminhamento.
- O Koito não substitui automaticamente a origem por “Koito” nesse fluxo.
- A escolha interna da coluna `client` não corresponde a uma reescrita do JSON encaminhado.
- Procurar apenas o texto “Koito” no cliente não é uma proteção confiável contra reimportação.

Essas conclusões se referem ao relay da ingestão analisada, não a uma garantia de que toda operação interna ou importação de arquivo seja encaminhada.

## 8. A API ListenBrainz permite leitura e envio

Base: `https://api.listenbrainz.org`.

| Operação | Endpoint |
| --- | --- |
| Enviar escutas | POST /1/submit-listens |
| Consultar histórico recente | GET /1/user/USUARIO/listens |
| Consultar reprodução atual | GET /1/user/USUARIO/playing-now |
| Consultar quantidade | GET /1/user/USUARIO/listen-count |
| Solicitar exportação | POST /1/export/ |
| Listar exportações | GET /1/export/list |
| Consultar andamento | GET /1/export/ID |
| Baixar exportação | GET /1/export/ID/download |

Exemplo:

```text
https://api.listenbrainz.org/1/user/USUARIO/listens?count=20
```

Parâmetros do histórico:

- `count`: quantidade; máximo documentado de 1.000 por chamada.
- `min_ts`: escutas com horário maior que o informado, limite exclusivo.
- `max_ts`: escutas com horário menor que o informado, limite exclusivo.
- Não combinar min_ts e max_ts na mesma chamada desse endpoint.
- A ordenação documentada é decrescente pelo horário da escuta.

A leitura do histórico não exige obrigatoriamente token; autenticação pode oferecer limites mais flexíveis. Exportações exigem token da própria conta.

É necessário identificar a aplicação com User-Agent e respeitar os cabeçalhos de limite e espera. Tokens devem ficar nas credenciais do aplicativo, fora de URLs e logs.

### Histórico completo versus atualizações

Para históricos grandes, a documentação recomenda exportação assíncrona ZIP. Para escutas recentes, consultas pequenas e atualizações incrementais, recomenda o endpoint de histórico.

Uma exportação pode ser solicitada, acompanhada até completar e baixada por API. É preciso reutilizar o mesmo trabalho em andamento e evitar solicitar outro a cada consulta.

A exportação não deve ser considerada uma fotografia atômica perfeita do histórico. Importações retroativas, alterações e exclusões exigem reconciliação adicional.

## 9. Importar somente escutas do Spotify

Não foi identificado filtro por serviço no endpoint documentado de histórico. A estratégia proposta é consultar o histórico e filtrar localmente no Koito.

Regra principal sugerida:

```text
track_metadata.additional_info.music_service == "spotify.com"
```

Campos auxiliares podem ser analisados quando o principal faltar, mas devem ser validados com exemplos reais da integração utilizada. Não foi feita essa validação na conta do usuário.

Política conservadora:

| Resultado | Ação proposta |
| --- | --- |
| Origem Spotify identificada | Candidata à importação |
| Outra origem identificada | Ignorar na sincronização restrita ao Spotify |
| Origem ausente ou ambígua | Não importar automaticamente; registrar para diagnóstico/revisão |
| Escuta já existente | Evitar nova inserção |

Não basta assumir que qualquer `submission_client` genérico ou link Spotify comprova uma reprodução no Spotify. Os campos são opcionais e declarados pelo cliente.

## 10. Fluxos desejados e prevenção de ciclos

Fluxos discutidos:

- Navidrome → Koito → ListenBrainz.
- Spotify → ListenBrainz → Koito.

A leitura do ListenBrainz deve usar um caminho de ingestão que **não acione o relay de volta ao ListenBrainz**. Essa regra é necessária mesmo quando existe filtro por Spotify.

Proteções propostas:

1. Configuração por perfil: usuário ListenBrainz, ativação e política de origem.
2. Estado persistente da sincronização.
3. Consultas com sobreposição temporal para atrasos e tentativas repetidas.
4. Verificação de escutas existentes antes de inserir.
5. Preservação de serviço e caminho de ingestão para diagnóstico.
6. Não reenviar ao relay registros obtidos do próprio ListenBrainz.
7. Reconciliação ocasional para dados retroativos.
8. Logs e contadores de importadas, ignoradas, duplicadas e falhas.

A detecção de duplicatas precisa preservar reproduções legítimas da mesma faixa. Perfil, timestamp e identificação da faixa são candidatos à comparação, mas a regra final precisa considerar o modelo atual do banco e testes reais.

As páginas chegam em ordem decrescente. Não se deve avançar o marcador apenas para o registro mais novo sem garantir que todas as páginas necessárias tenham sido processadas; isso pode pular escutas em períodos de alto volume.

A frequência da sincronização ainda não foi definida. Ela deve respeitar limites de API, quantidade de perfis e atraso aceitável. Escutas do Spotify podem levar alguns minutos para chegar ao ListenBrainz.

## 11. Situação das decisões

| Item | Situação |
| --- | --- |
| Go backend + SPA React | Confirmado no código analisado |
| Um servidor Go no contêiner final | Confirmado no Dockerfile analisado |
| SQLite | Confirmado |
| Importador ListenBrainz | Confirmado |
| Coluna client e preferência por media_player | Confirmado no código |
| Relay do corpo original | Confirmado no handler analisado |
| Leitura de escutas por API ListenBrainz | Confirmado na documentação |
| Filtro de Spotify no servidor ListenBrainz | Não identificado no endpoint consultado |
| Filtragem local por music_service | Proposta viável, dependente dos metadados |
| Migração para templates Go | Avaliada como viável; não implementada |
| Sincronização recorrente ListenBrainz → Koito | Proposta; não implementada |
| Metadados reais da conta do usuário | Ainda não consultados |
| Ganhos de desempenho | Não medidos |

## 12. Fontes e arquivos de referência

### Repositório

- [Koito-SUM](https://github.com/ebrgr/TuneTrail)
- [Base inicial analisada](https://github.com/ebrgr/TuneTrail/tree/a936c1c1b050db486a67c6d429a4c0960d83f790)
- [Configuração React Router](https://github.com/ebrgr/TuneTrail/blob/a936c1c1b050db486a67c6d429a4c0960d83f790/client/react-router.config.ts)
- [Rotas do servidor](https://github.com/ebrgr/TuneTrail/blob/a936c1c1b050db486a67c6d429a4c0960d83f790/engine/routes.go)
- [Dockerfile](https://github.com/ebrgr/TuneTrail/blob/a936c1c1b050db486a67c6d429a4c0960d83f790/Dockerfile)
- [Importador ListenBrainz — branch principal](https://github.com/ebrgr/TuneTrail/blob/main/internal/importer/listenbrainz.go)
- [Ingestão e relay — branch principal](https://github.com/ebrgr/TuneTrail/blob/main/engine/handlers/lbz_submit_listen.go)
- [Persistência de escutas — branch principal](https://github.com/ebrgr/TuneTrail/blob/main/internal/db/sqlite/listen.go)

Links para main podem mudar. A conferência posterior sobre client foi feita durante a conversa, sem registrar um segundo hash no relatório original.

### Documentação oficial

- [Importação no Koito](https://koito.io/guides/importing/)
- [Configuração de scrobbling](https://koito.io/guides/scrobbler/)
- [API ListenBrainz](https://listenbrainz.readthedocs.io/en/latest/users/api/)
- [Endpoints de escutas](https://listenbrainz.readthedocs.io/en/latest/users/api/core.html)
- [Formato JSON e origem](https://listenbrainz.readthedocs.io/en/latest/users/json.html)
- [Exemplos de uso](https://listenbrainz.readthedocs.io/en/latest/users/api-usage.html)
- [Exportação de histórico](https://listenbrainz.readthedocs.io/en/latest/users/api/export.html)
- [Intervalos de atualização](https://listenbrainz.readthedocs.io/en/latest/general/data-update-intervals.html)

