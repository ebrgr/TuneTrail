# Escopo proposto para a ISSUE-104

A especificação deve descrever operações em uso cujo contrato o projeto pretende manter. O [inventário por método e caminho](ROTAS.md) registra os handlers, a política atual e a evidência de uso. A ISSUE-104 também propõe métricas Prometheus. `/metrics` ainda não está registrado nesta revisão.

## Documentar

| Operações | Contrato a registrar |
|---|---|
| `POST /apis/listenbrainz/1/submit-listens` | Compatibilidade de ingestão ListenBrainz. Discriminador `listen_type`, payload em lote ou unitário conforme tipo, resposta JSON, limites do lote e corpo, e resultados de erro. Diferenciar `playing_now` do histórico persistido e explicar ACK conforme o modo de ingestão. |
| `GET /apis/listenbrainz/1/validate-token` | Validação da mesma chave de ingestão e identidade resolvida do perfil. Cobrir sucesso, credencial ausente, inválida ou revogada e falha de armazenamento. |
| `GET /apis/web/v1/health` | Prontidão pública, 200 sem corpo e 503 quando indisponível. Consumida pelo HEALTHCHECK e pelos scripts. |
| `GET /apis/web/v1/profiles` e `GET /apis/web/v1/profiles/{slug}` | Diretório e metadados de perfil. Explicar a diferença entre conhecer os metadados e poder ler o histórico de um perfil protegido. Consolidar campos estáveis na ISSUE-145. |
| Leituras com slug de `listens`, `listen-activity`, `first-activity`, `now-playing`, `stats`, `search`, `summary` e `top/{tracks,albums,artists}` | Candidatas ao contrato JSON de leitura da ISSUE-145. Fixar paginação, período, fuso, semântica do ranking e estados vazios. `now-playing` representa estado efêmero, sem garantia de persistência nem streaming. |
| Leituras com slug de `artist/{id}`, `album/{id}`, `track/{id}` e respectivos sufixos `interest` | Candidatas ao contrato de leitura quando a ISSUE-145 estabilizar identificadores, visibilidade, filtros e formatos. Confirmar público para os subrecursos `aliases` e `artists` antes de incluí-los. |
| `GET /apis/web/v1/profiles/{slug}/export` | Preservar e documentar o anexo de histórico nativo v2. A publicação da operação na OpenAPI depende de definir o acesso para consumidores externos. Hoje exige sessão humana com acesso ao histórico, dono ou root. |
| `GET /image/profile/{profile_id}/{image_id}/{filename}` e `GET /image/{image_id}/{filename}` | Referências binárias retornadas por clientes e páginas. Registrar em seção complementar de integração e explicitar variante legada, vínculo com o perfil, visibilidade, cache e 404. |

## Manter internas

- Páginas Go e fragmentos, `/language`, `/account*`, `/admin*`, `/{slug}/manage/*`, mounts de assets e o executável de prévia. Permanecem contratos de navegação e formulários. A exportação HTML compartilha o formato do anexo com a versão JSON.
- Sessão web JSON em `/login`, `/logout`, `/user` e `/config` sob `/apis/web/v1`. As operações de conta, chaves, avatar, gestão do perfil, escutas manuais e mutações do catálogo dependem de sessão humana. Criação e exclusão de perfis, permissões de relay e catálogo global exigem root.
- Diagnóstico em `/admin/health`, revisão e reprocessamento, configurações e todos os endpoints de gestão de relay. O relay em si é consumidor de destinos ListenBrainz e pode ser descrito no guia da integração.
- Aliases JSON sem slug e redirects de páginas antigas. Registrar a migração para destino com slug e o 409 dos aliases JSON em servidor multiperfil na documentação de compatibilidade.
- Importação multipart permanece ação de gestão. Documentar os formatos de arquivo suportados e separar essa decisão da ingestão ListenBrainz.

## Decisões antes de publicar

- **ISSUE-121.** Modelar a chave atual como cabeçalho `Authorization: Token <chave>` nas duas operações ListenBrainz. A chave resolve um perfil no servidor. Leituras privadas, exportação e operações de sessão seguem políticas próprias. Caso sejam criados escopos, definir separadamente ingestão, estatísticas e histórico, assim como revogação, seleção do perfil, visibilidade e precedência em relação a cookies. Resolver efeitos de proxy, TLS e subcaminho sobre URL base, CORS e cookies.
- **ISSUE-145.** Fixar, para aplicativos nativos, o conjunto de leituras estáveis, tipos de erro, paginação, versionamento, token por dispositivo, revogação, renovação e descoberta segura da instância. Garantir testes com visitante, dono, outro usuário, root e dispositivo. Perfis privados ocultam o histórico com 404. O diretório público e as imagens precisam de exemplos próprios de visibilidade.
- **ISSUE-149.** Decidir se exclusões globais, purge, demais mutações JSON do catálogo e undo de merge seguem acessíveis a automações. O undo tem uso documentado. Nenhuma dessas operações deve ganhar estabilidade pública por aparecer na árvore Chi.
- **ISSUE-104.** Registrar erros e proteção de origem por família, o HEAD herdado e a semântica de 401/403/404/409 nos testes de contrato. Para Prometheus, usar o padrão da rota como dimensão de baixa cardinalidade e manter caminhos concretos, títulos, perfis, tokens e demais identificadores fora dos rótulos.
