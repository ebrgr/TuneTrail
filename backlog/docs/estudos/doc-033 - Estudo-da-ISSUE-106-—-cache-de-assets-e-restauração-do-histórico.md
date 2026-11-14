---
id: doc-033
title: Estudo da ISSUE-106 — cache de assets e restauração do histórico
type: other
created_date: '2026-11-14T20:15:00Z'
updated_date: '2026-11-14T20:28:00Z'
tags:
  - estudo
---
**Tarefa:** ISSUE-106 · **Base:** `main` em `3bcc93e`, TuneTrail 0.11 · **Data:** 14/11/2026.

Estudo de implementação, sem mudança no produto. A ISSUE-066 está aceita e publicada localmente; a 106 deve usar os arquivos externos e o snapshot dessa versão. As recomendações abaixo são propostas para a resolução, não decisões de arquitetura aceitas pelo usuário. Não há mudança de banco nem migração necessária.

## Resultado e escopo

Resolver a 106 em uma branch funcional e publicar uma versão depois do aceite. Reutilizar o snapshot, acrescentar um catálogo de assets com hashes individuais, aplicar cache HTTP coerente e corrigir o ciclo de vida das páginas restauradas pelo histórico. A tarefa não é só uma troca de cabeçalhos: existe um controlador abortado no `live.js` que precisa ser recriado.

Ficam fora: reorganizar componentes da 067, dividir CSS da 068, fontes locais/estilos da 070, navegação parcial da 075, service worker da 076 e atualização de escutas por SSE. Fontes continuam como estão até a 070; a infraestrutura de hash deve aceitar WOFF2 quando essa tarefa os adicionar ao manifesto.

## Diagnóstico confirmado no disco

| Local | Situação na 0.11 | Consequência |
| --- | --- | --- |
| `internal/webassets/assets.go` | `Current` carrega uma geração via `sync.Once`; `Snapshot.Hash` é agregado; arquivos servidos vêm de memória | Base pronta, mas falta identidade de cada representação publicada |
| `internal/webui/assets.go` | `StaticHandler` usa FileServer com modtime zero; sem ETag/cache; `nosniff` e MIME explícito | Requisições repetidas transferem os recursos novamente |
| `internal/webui/preview.go` | `themes.css` também é gerado no handler da prévia | Há duplicação a substituir por bytes derivados determinísticos |
| `assets/css/admin.css` | Cinco `url('/web-assets/icons/…')` | Hash só no `<link>` não versiona essas dependências |
| `assets/js/ui.js` | Busca `themes.json` concatenando `data-assets` | O fetch também precisa receber URL versionada |
| `internal/webui/handlers.go:render` | Sucesso e erros públicos compartilham `no-store` | Aplicar política por status, inclusive falha de renderização |
| `internal/adminui/admin.go:New` | Wrapper `adminweb.NoStore` antes do dispatch | Preservar também em login, gestão e confirmações fora de `/admin` |
| `internal/i18n/i18n.go:Wrap` | `Content-Language`, `Vary: Accept-Language`; idioma também depende de cookie HttpOnly | Somar `Vary: Cookie` no HTML público sem apagar o Vary existente |
| `assets/js/live.js` | Um AbortController, abortado por `pagehide` com `once:true`; DELETE captura esse controlador; AbortError retorna com botão desabilitado | Ao restaurar a página, a ação pode falhar imediatamente e ficar travada |

### Medição local

Servidor e SQLite descartáveis, dados fictícios, nenhum acesso à instalação operacional. Cabeçalhos em `backlog/evidencias/issue-106/http-base-0.11.json`:

- `ui.css`: 28.634 bytes; `themes.css`: 4.423; `theme.js`: 2.366; `themes.json`: 4.618. Todos sem Cache-Control, ETag ou Last-Modified e com `nosniff`.
- HTML público e `/admin`: `Cache-Control: no-store`; Vary contém somente Accept-Language.
- Chrome **157.0.8103.51**, headless, sessão anônima: primeira e segunda navegações transferiram os seis recursos locais novamente, **48.737 bytes de conteúdo / 50.537 de transferSize por navegação**. Sem interceptação `page.route`, sem desabilitar o cache HTTP, removendo a flag padrão que desabilita bfcache. Fontes externas foram bloqueadas pelo CDP, sem alterar os arquivos nem a política de cache dos assets.
- Voltar: `pageshow.persisted=false`, novo documento. O navegador informou `response-cache-control-no-store` e `response-cache-control-no-store-with-js-network-request`, além de razão mascarada. Esse é o resultado desta sessão, não uma afirmação universal sobre Chrome. Evidência: `chrome-base-0.11.json`.
- Firefox real ainda **não medido**. O Firefox instalado não foi iniciado e não há pacote Firefox do Playwright no cache local. Não afirmar compatibilidade ou restauração nele antes de executar o protocolo de aceite.

Reprodução: `make api.build`, depois `PLAYWRIGHT_MODULE=/caminho/node_modules/playwright python3 backlog/evidencias/issue-106/medir-base.py --output /tmp/medicao-106-nova`. O diretório deve ser novo. `CHROME_PATH` seleciona o Chrome. Os arquivos só guardam cabeçalhos selecionados, tamanhos e eventos; não guardam senhas/cookies/sessões. O protocolo exige recompilar antes de medir; o script usa VERSION para identificar os arquivos `http-base-<versão>.json` e `chrome-base-<versão>.json`, e também registra a versão no JSON HTTP. A medição repete a versão compilada nesse protocolo, não uma simulação da implementação futura.

## Duas formas de cache, com contratos diferentes

Cache HTTP guarda respostas; bfcache restaura documento e heap JavaScript. `private, no-cache` permite armazenamento privado com validação antes da reutilização HTTP. Não significa ausência de armazenamento e não garante bfcache. A documentação específica do Chrome permite alguns documentos `no-store` no bfcache desde 2025; o diagnóstico antigo da issue precisava dessa ressalva. Medir navegador/versão e `pageshow.persisted`, não deduzir o resultado só pelo cabeçalho. [RFC 9111](https://www.rfc-editor.org/rfc/rfc9111.html), [Chrome](https://developer.chrome.com/docs/web-platform/bfcache-ccns), [ciclo de vida do bfcache](https://web.dev/articles/bfcache).

## Catálogo e hashes propostos

Manter a leitura confinada, manifesto, limites, licenças privadas e uma geração por processo da 066. Nenhum caminho de disco vem da requisição. Separar arquivos privados, como HTML e LICENSE, das representações servíveis. Não tornar todo o root público.

Preparação na inicialização:

1. Carregar/validar o snapshot da 066.
2. Validar paletas e gerar `themes.css` uma única vez, com ordenação determinística. Centralizar o gerador hoje duplicado. A regra comum pertence a `internal/webassets`, sem importar `webui` e criar ciclo. Os view models públicos continuam consumindo a mesma geração.
3. Calcular SHA-256 completo dos bytes finais das folhas (JS, JSON, imagens, SVGs, futuras fontes).
4. Reescrever referências internas do CSS para as URLs com hash dessas folhas; só depois calcular o hash do CSS final. Alterar um ícone deve mudar o hash dele e o de `admin.css`, mas não os de arquivos independentes. Editar HTML não deve invalidar CSS/JS inalterados.
5. Publicar catálogo e metadados de forma imutável. Aplicar os limites também aos bytes derivados/expansão de URLs; não permitir que a transformação burle os limites da 066.

O hash agregado continua útil como identidade de geração, combinado à versão para detectar troca do produto. Não usá-lo como versão de todos os arquivos. ETag forte: `"<sha256hex>"`, calculado sobre a representação realmente enviada. Sem timestamp artificial. Compressão não entra agora; se for adicionada depois, precisa de identidade por codificação e Vary apropriado.

### CSS: implementação mínima suficiente

Hoje existem apenas CSS→folha. Não implementar DAG, armazenamento de releases nem resolvedor genérico de dependências agora. Usar reconhecimento léxico do CSS, respeitando strings, comentários e escapes; uma regex global de `url()` não é o contrato. Fragmentos (`url(#…)`), dados e referências externas permanecem intactos. Referências ao namespace de assets devem resolver em uma entrada pública conhecida; erro/missing recusa startup.

Recusar `@import` real e dependência CSS→CSS no startup nesta etapa. A 068 concatena as camadas antes dessa transformação; se precisar de imports, deverá definir a nova extensão e seus ciclos. Não modificar `url(#…)` dos SVGs inline dos templates.

Emitir URLs **relativas ao caminho lógico publicado do CSS**, não ao diretório físico `assets/css/`. Ex.: `icons/Music2.svg?v=<hash>`. Assim os mesmos bytes CSS e o mesmo hash funcionam sob `/web-assets/` e `/preview-assets/`; não gerar CSS diferente por prefixo. Preservar fragmentos após a query de hash. Para referências internas, recusar query preexistente na fonte nesta etapa: o handler só aceita a query canônica `v`; não produzir URLs com parâmetros que ele recusará. Dados e URLs externas permanecem intactos.

## Helpers e integração

Contrato sugerido, com nomes finais a definir na implementação:

```go
// Private immutable bytes; a fresh reader is created per request.
type Asset struct { /* bytes, digest, content type */ }
func (s *Snapshot) Asset(name string) (Asset, bool)
func (s *Snapshot) AssetURL(prefix, name string) (string, error)
```

`asset` é registrado antes do ParseFS nos conjuntos público/admin e na prévia, com prefixo fixado pelo servidor. Ex.: `{{asset "ui.css"}}` produz `/web-assets/ui.css?v=<hash>`. O nome é chave lógica de catálogo. `asset` não recebe caminhos de disco, prefixos arbitrários de usuário ou nomes dinâmicos.

Ampliar a validação AST dos templates para conferir chamadas `asset` com literal conhecido e recusar argumentos não literais/entradas privadas ou ausentes no startup, inclusive em ramos não executados. Preservar os namespaces separados e o comportamento de erro da 066.

O fetch de paletas passa a usar `data-themes-url="{{asset "themes.json"}}"`, tanto em produção quanto na prévia; não concatenar um caminho sem hash no JS. Atualizar todos os shells, inclusive favicon. Conservar a ordem de CSS/scripts e o script de tema antes da pintura.

## Contrato HTTP proposto

| Recurso/requisição | Política |
| --- | --- |
| Asset existente, GET/HEAD, uma única query canônica `v` igual ao hash | `public, max-age=31536000, immutable`, ETag forte |
| Asset existente, sem query | `no-cache`, mesmo ETag; compatibilidade com URLs fixas |
| Condicional válida no asset resolvido | 304 sem corpo, preservando ETag e Cache-Control |
| Hash antigo/errado/vazio/duplicado, query inválida/extra ou arquivo ausente | 404, `no-store`; nunca responder bytes novos sob hash velho |
| Método diferente de GET/HEAD | 405 com Allow e `no-store` |
| HTML público 200 GET/HEAD | `private, no-cache`, Vary incluindo Cookie e Accept-Language; sem novo validador HTML |
| HTML 4xx/5xx/falha de render | `no-store` |
| Admin/login/gestão/confirmar, ações, CSRF, `/language`, APIs protegidas | Preservar `no-store` |

Resolver a entrada/versão antes de avaliar condicionais. Usar `http.ServeContent` com ETag definido, reader novo e modtime zero para GET/HEAD, listas/`*` em If-None-Match e Range/If-Range. Não escrever um parser HTTP condicional paralelo. Em erros internos dessa função (p.ex. 412/416), garantir `no-store` e remover validadores/immutable antes de escrever o status: ServeContent pode remover cabeçalhos de cache no erro. Não habilitar globalmente a opção de preservar todos os cabeçalhos em erros. [Documentação oficial do Go](https://pkg.go.dev/net/http#ServeContent).

### Aliases e recursos fora do prefixo principal

Proposta explícita: referências dos shells passam ao helper canônico com hash. `/public/*` e `/favicon.svg` continuam como aliases legados, com URL fixa `no-cache`+ETag, sem cache longo. O `site.webmanifest` legado mantém seu JSON e ícones `/public/…` fixos, ambos revalidáveis; não prometer que seu grafo é imutável. O trabalho PWA da 076 pode versionar esse conjunto depois. Os mesmos bytes não ganham uma política contraditória só por passar por `StaticHandler`.

A prévia usa o catálogo para estáticos e temas sob `/preview-assets/`. Capas fictícias geradas por ID e fotos de `/image/…` não são arquivos estáticos do catálogo e não recebem `immutable` por esta tarefa. APIs, imagens de usuários e integrações externas não entram no cache público dos assets.

## Retenção entre releases: recomendação e limite

**Proposta recomendada para a 106:** servir somente a geração atual. Hash anterior desconhecido responde 404/no-store. Não escrever cache persistente no diretório de dados nem manter uma pasta mutável de assets. Nenhuma nova variável, migração ou retenção ilimitada é necessária.

Uma aba/documento antigo pode referenciar recursos tardios ou removidos do cache que o processo novo não possui. Isso inclui a situação em que o próprio JS antigo não está mais disponível: o mecanismo de recuperação em JS não poderá executar. Nesse caso é necessário recarregar o documento. A proposta preserva coerência de conteúdo, mas **não garante disponibilidade de URLs antigas nem continuidade de abas por atualizações**. Registrar isso em DEPLOYMENT.md, incluindo caches/proxy que precisam preservar a query. Não anunciar suporte a rolling deploy com gerações diferentes.

A alternativa é distribuir e servir representações finais históricas, por hash, com retenção, limites, expiração e compatibilidade de formato entre versões explicitamente definidos. Só manter a pasta-fonte anterior não garante reproduzir exatamente o CSS transformado com outro código. É trabalho adicional; não incluí-lo implicitamente. A escolha pela geração atual precisa ser confirmada antes da implementação, junto com as demais propostas deste estudo.

## Voltar/avançar e estado da aplicação

HTML público restaurado conserva dados, filtros e posição como estavam; não atualizar todas as estatísticas por polling nem recarregar toda restauração indiscriminadamente.

No `live.js`, separar ativação/desativação:

- Em `pagehide`, abortar operações daquela ativação e remover/ocultar os controles administrativos antes de suspender. Não limitar o handler ao primeiro pagehide.
- Na primeira carga e em `pageshow.persisted`, criar controlador novo e consultar `/apis/web/v1/user` com `cache:'no-store'`, credenciais same-origin e prazo limitado. Só reintroduzir controles após confirmação de sessão administrativa.
- Impedir duplicação de listeners/células. Handlers devem usar o controlador ativo, não o que foi capturado antes de suspender. Respostas antigas/abortadas não podem reativar uma sessão obsoleta.
- `finally` reabilita o estado aplicável dos botões; aborto não deixa botão travado. Uma mutação abortada pode ter sido aplicada no servidor: nunca repetir POST/DELETE automaticamente.
- Em falha de rede, sessão expirada ou logout, manter a informação pública e os controles de mutação ausentes. A autorização continua sendo verificada pelo servidor em cada ação.

Para mudança de release/idioma, a proposta é HEAD em segundo plano **somente em `pageshow.persisted`**, com timeout, credenciais same-origin e `cache:'no-store'`. Comparar `Content-Language` com o documento e `X-TuneTrail-UI-Generation` com `data-ui-generation` (versão+hash do snapshot). Se mudou ou o recurso deixou de existir, recarregar. Ausência de cabeçalho opcional, erro transitório ou preview sem catálogo não deve provocar loop de reload. Comparar também cookie de fuso com `data-timezone` quando não houver fuso explícito; preservar a escolha por query. Sincronizar tema com a preferência atual.

Esse HEAD reutiliza a semântica de GET, inclusive perfil inexistente. Ele **pode executar SQL/renderização**, então não prometer economia de consultas. O ganho principal é a exibição imediata do documento e o cache dos assets. Medir o custo antes de introduzir um endpoint extra; não criar agora um endpoint de contexto só para evitar esse HEAD.

No `admin.js`, `pageshow.persisted` recarrega para revalidar sessão e regenerar formulários. Ocultar conteúdo/controles sensíveis em pagehide para não depender de no-store como garantia de descarte. Falha offline deve manter operações indisponíveis, sem revelar dados novos ou repetir envios. No-store pode permitir bfcache no Chrome; uma sessão expirada sem mudança de cookie exige esse cuidado também.

## Arquivos e sequência para resolver

1. `internal/webassets/assets.go` e novos arquivos de catálogo/temas/reescrita: separar fonte privada/representação pública, derivar bytes, hashes, MIME e leitores; limites e testes.
2. `internal/webassets/templates.go`: validar chamadas asset literais antes da partida.
3. `internal/webui/assets.go`, `preview.go`, `engine/routes.go`: handler compartilhado, condicionais, aliases e prévia; eliminar geração repetida dos temas.
4. `internal/webui/renderer.go`, `internal/adminui/admin.go`, `assets/templates/public/{live,page}.html`, `assets/templates/admin/shell.html`, `assets/js/ui.js`: helper e URLs versionadas, sem unificar os componentes nesta issue.
5. `assets/css/admin.css`: URLs das folhas reescritas na geração; não gravar hashes à mão na fonte.
6. `internal/webui/handlers.go` e contexto de renderização: política por status, Vary e identidade de geração.
7. `assets/js/live.js`, `theme.js` se necessário e `admin.js`: ciclo de restauração e sessão; testes com ciclos repetidos.
8. Testes de `internal/webassets`, `internal/webui`, `internal/adminui`, `engine/web_ui_default_027_test.go`, validação e2e da distribuição, DEPLOYMENT.md, README de desenvolvimento, CHANGELOG Unreleased.
9. Atualizar a referência de renderização de forma revisável: a 106 muda URLs HTML, CSS transformado e JS. O overlay **somente HTML** da 066 não autoriza esses novos bytes CSS/JS; criar base da 106 ou evolução explícita do gate, preservando as bases históricas. Não substituir hashes por curingas nem aceitar diferenças automaticamente.
10. Reviewer, `make check` sem cache, verificação navegadores/proxy, Aguardando aceite; somente depois de aceite/publicação autorizados, `make issue.publish ISSUE=106`, uma versão. Push separado.

A 068 reutiliza o catálogo após concatenar CSS; a 070 adiciona folhas WOFF2 e mantém licenças privadas; a 075 herda o ciclo de pageshow. A 109 deve reutilizar a verificação de papel e não acrescentar uma segunda busca concorrente de `/user`.

## Verificação exigida na implementação

- Hash estável entre partidas/ordenação de manifesto; mudança de arquivo ou folha dependente; bytes servidos iguais aos usados no hash; nenhum acesso ao disco por request; limites dos derivados.
- Templates: asset ausente/privado/nome não literal falha na inicialização. CSS com referência ausente, import ou dependência CSS→CSS falha, sem transformar dados/URLs externas/fragmentos.
- HTTP real e unitário: GET/HEAD, 200/304/206, If-None-Match igual/lista/fraco/`*`, If-Range, hash errado/antigo/duplicado/query extra, 404/405, 412/416 e corpo ausente no HEAD/304. Política de erro não pode herdar immutable.
- Política por rota/status, dois perfis, cookies, dois idiomas e fusos; login/gestão/CSRF continuam protegidos.
- Chrome e Firefox de versões anotadas: duas navegações distintas com cache habilitado, sem `page.route`; sem transferir novamente corpos CSS/JS já frescos. 304 é revalidação, não prova de cache imutável sem ida ao servidor. [Playwright: routing desabilita o cache HTTP](https://playwright.dev/docs/api/class-page#page-route).
- Bfcache: `pageshow.persisted===true`, identidade do documento, scroll/filtros preservados e nenhuma nova navegação do documento principal para o caminho estável. HEAD/auth em background são registrados separadamente. O harness não pode manter a flag que desabilita bfcache. [Opções de lançamento](https://playwright.dev/docs/api/class-browsertype#browser-type-launch-option-ignore-default-args).
- Login/logout em outra aba e expiração de sessão sem mudança de cookie; duas ou mais sequências ir/voltar; botões operacionais só com sessão validada, sem callbacks duplicados; abortos e falhas de rede.
- Também medir navegação de histórico com `persisted=false` e `type=back_forward`, distinguindo documento reaproveitado de requisição nova. Vary ajuda o cache HTTP, mas não deve ser presumido como garantia sobre todo mecanismo de histórico. O HEAD proposto só cobre `persisted=true`; o outro caminho mantém o limite de gerações antigas e exige comprovação de idioma/fuso/sessão. A evidência de themes.json vindo do cache não prova reaproveitamento do HTML. Confirmar que 404 no fetch de paletas mostra erro sem inutilizar a página (temas embutidos na geração continuam disponíveis).
- Reinício com release diferente e URL antiga sem cache local: 404/no-store e limite de recuperação documentado. Sem prometer retenção histórica.
- HTTP local e instalação equivalente atrás de proxy HTTPS; o proxy não remove query, não ignora os cabeçalhos e não aplica cache público ao HTML privado. Firefox/HTTPS não foram medidos neste estudo e continuam gates da implementação.

## Parecer e decisões antes de implementar

O Advisor inicial respondeu **REPROVADO COM RESSALVAS**. Incorporados: ciclo de AbortController, status de erro, aliases/manifest, literal asset validado no startup, sem DAG desnecessário e limite da geração atual. O parecer original é preservado em `backlog/evidencias/issue-106/advisor-inicial.md`. Ele não conseguiu conferir regras no AGENTS.md local, que contém só uma referência; o agente principal aplicou as instruções completas fornecidas nesta conversa e não alterou esses arquivos.

Propostas a confirmar no plano de implementação: geração atual apenas com limite explícito; aliases/manifest legados revalidáveis; HEAD somente em restauração persistida aceitando seu custo de SSR. Nenhuma dessas escolhas representa aceite da implementação. A issue permanece Planejada, com critérios ainda desmarcados.

A reavaliação do Advisor ficou **APROVADA, sem lacunas bloqueantes**, em `backlog/evidencias/issue-106/advisor-final.md`. Incorporadas as recomendações textuais sobre o caminho de histórico sem bfcache e a identificação reproduzível da versão nas evidências. O parecer aprova o estudo para apresentação, não substitui a confirmação das propostas nem o aceite do produto.
