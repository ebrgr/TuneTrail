# Parecer do Advisor (reavaliação): APROVADO

Li só os seis arquivos autorizados. Não alterei nenhum arquivo, não deleguei a outro agente e não implementei nada de runtime. A issue continua Planejada, com os ACs desmarcados.

## Ressalvas do parecer inicial

| Ressalva | Situação no doc-033 |
| --- | --- |
| AbortController abortado no `live.js` | **Resolvida.** O estudo cria um controlador por ativação, tira o `once` do `pagehide`, faz o `finally` reabilitar os botões, não repete POST/DELETE e ignora respostas antigas. Também cobre o reload do `admin.js` com ocultação no `pagehide`. |
| Geração atual e limite de recuperação | **Resolvida.** O limite está explícito: URLs antigas não ficam disponíveis, o JS antigo pode ter sido despejado do cache e não há rolling deploy. Vai para o DEPLOYMENT.md, e a alternativa com histórico de releases está separada e não fica implícita. |
| Aliases e manifest | **Resolvida.** `/public/*`, `/favicon.svg` e `site.webmanifest` ficam fixos com `no-cache`+ETag. Os shells passam a usar o helper e o versionamento do PWA fica para a 076. |
| Erros do SSR | **Resolvida.** A política é decidida pelo status: 200 recebe `private, no-cache` e 4xx/5xx/falha de render recebem `no-store`. O `Vary: Cookie` é somado ao valor existente, sem substituí-lo. |
| Literal de `asset` validado | **Resolvida.** A validação pela árvore do template cobre também ramos que não executam e recusa argumento não literal, nome privado ou ausente. |
| Sem DAG (YAGNI) | **Resolvida.** O CSS é lido por análise léxica, `@import` e CSS→CSS são recusados, e a ordem de dependências fica para a 068. |

## Pontos novos do plano consolidado

- **CSS com URL relativa:** é correto. `icons/X.svg?v=…` dentro de `/web-assets/admin.css` resolve para `/web-assets/icons/X.svg`, e o mesmo vale sob `/preview-assets/`. Os bytes e o hash do CSS ficam iguais nos dois prefixos, e a query não altera a resolução relativa.
- **Erro para query que já existe na fonte:** é coerente com o handler, que só aceita um `v`.
- **412/416 do `ServeContent`:** está tratado como requisito. Há um detalhe de implementação na seção de opcionais.
- **HEAD só em `persisted`, sem prometer economia de SQL:** é coerente e cita o comportamento herdado do GET.

## Fatos conferidos nas evidências

- **Bytes de conteúdo:** 4.423 + 2.366 + 28.634 + 4.855 + 3.841 + 4.618 = **48.737**. Bate com o estudo.
- **transferSize:** a soma dá **50.537**. Bate.
- **Segunda navegação:** os seis recursos voltaram a ser baixados. Bate.
- **Voltar:** `persisted:false` e `sameDocument:false`, com as razões `masked`, `response-cache-control-no-store` e `…-with-js-network-request`. Bate.
- **Harness:** usa `ignoreDefaultArgs` para tirar a flag que desabilita o bfcache, não usa `page.route` e bloqueia as fontes pelo CDP e pelo host-resolver. Bate.
- **Cabeçalhos HTTP:** os quatro assets saem sem Cache-Control, ETag e Last-Modified e com `nosniff`. O HTML e o `/admin` saem com `no-store` e `Vary: Accept-Language`. Bate.
- **Firefox e proxy:** aparecem corretamente como não medidos e como gates futuros, sem alegar compatibilidade.

## Lacunas bloqueantes

Nenhuma.

## Ajustes textuais recomendados antes de apresentar (não bloqueiam)

1. **Voltar sem bfcache, com HTML vindo do cache HTTP.**
   - Ao trocar `no-store` por `private, no-cache`, o Chrome pode reaproveitar o HTML do cache numa navegação de histórico sem revalidar. Nesse caso `persisted` vale false e o HEAD de geração não roda.
   - A própria evidência sugere isso: na fase `back`, o `themes.json` veio com `fromDiskCache:true` e 0 bytes, mesmo sem cabeçalhos de cache.
   - O `Vary: Cookie` deve forçar nova busca quando o idioma, o fuso ou a sessão mudam. Isso ainda precisa ser medido nos navegadores, não pode ser presumido.
   - Uma troca de release sem mudança de cookie cai no limite já documentado.
   - **Sugestão:** acrescentar em "Verificação exigida" a medição desse caminho e anotar que ele está coberto pelo limite e pelo Vary, não pelo HEAD. Também vale exigir que o `ui.js` lide com 404 do `themes.json` sem quebrar a página.
2. **Reprodutibilidade das evidências.**
   - O `medir-base.py` grava `http-base.json` e `chrome-base.json` sem o campo `version`.
   - Os arquivos commitados se chamam `*-0.11.json`, e o JSON HTTP traz `"version": "0.11"`.
   - **Sugestão:** registrar que o nome e o campo foram ajustados à mão, ou fazer o script emitir os dois.

## Opcionais (fora do escopo mínimo)

- **412 e 416 no `ServeContent`:** não falham do mesmo jeito.
  - O 412 sai direto da checagem de pré-condições e mantém os cabeçalhos já definidos, inclusive `immutable`.
  - O 416 passa pelo caminho de erro, que remove `Cache-Control` e ETag nas versões recentes do Go.
  - Um wrapper de `WriteHeader` que aplique `no-store` para status ≥ 400 cobre os dois casos, e o teste deve exercitar ambos. Isso já cabe no texto atual; aqui é só uma nota para a implementação.
- **`public` em `/preview-assets`:** são os mesmos bytes estáticos e não sensíveis. Dá para manter, mas vale uma frase no DEPLOYMENT dizendo que proxies podem armazená-los.

O estudo está pronto para ser apresentado e, depois que o usuário confirmar as três propostas, para implementação na branch da issue. As três propostas são: servir só a geração atual com o limite descrito, deixar aliases e manifest legados revalidáveis, e fazer HEAD só em `persisted` aceitando o custo de SSR.
