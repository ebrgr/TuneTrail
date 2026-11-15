# Desenvolvimento local

Koito registra escutas e fornece uma API compatível com ListenBrainz e uma interface web Go. Não é necessário Docker nem PostgreSQL para desenvolver. O esquema consolidado do SQLite é a versão 3; as migrações são executadas na inicialização.

## Preparação e execução

Instale Go na versão indicada por `go.mod` e `make`. A aplicação e seu processamento de imagens compilam com `CGO_ENABLED=0`, sem compilador C ou libvips. Node.js é necessário para os testes dos pequenos scripts da interface e para o site de documentação. O detector de corridas (`make api.race`) usa CGO e exige compilador C.

```sh
make setup       # módulos Go, Delve local e .env se ainda não existir
make api.dev     # API e interface em http://127.0.0.1:4110
make check       # Go sem CGO, vet, testes da interface e suíte e2e (make e2e)
make e2e         # só as jornadas de ponta a ponta, em dados fictícios descartáveis
make api.race    # opcional: detector de corridas, requer compilador C
make build       # binário ./tunetrail com templates e assets incorporados
make multiuser.check  # só a jornada multiusuário (também incluída em make e2e)
```

A raiz permite selecionar um perfil, `/<slug>` abre sua página e `/admin` acessa a administração. O `.env` é carregado pelo Makefile e pela configuração do VS Code, não por `go run` diretamente. Dados locais ficam em `.local/koito`; não os use para testes destrutivos. Consulte `docs/src/content/docs/guides` para importação e configuração.

O React foi removido. `KOITO_WEB_UI` pode ser omitida ou definida como `go`; o antigo valor `react` impede a inicialização com mensagem explícita. Para voltar a uma versão antiga é necessário planejar separadamente a compatibilidade do banco e dos dados posteriores ao backup.

## Depuração

No VS Code, instale a extensão Go e use **TuneTrail: Desenvolvimento Go (localhost:4110)** com F5. O Delve inicia a aplicação e abre o navegador quando ela estiver pronta. Pare uma instância anterior se a porta estiver ocupada. `make api.debug` inicia o Delve no terminal. Reinicie o servidor após alterar código, templates ou assets incorporados.

## Estrutura e validação

`cmd/api` contém a entrada; `engine` concentra rotas e handlers; `internal/webui` e `internal/adminui` implementam as páginas públicas e administrativas; `internal/db/sqlite` e `db/migrations_sqlite` contêm persistência e migrações. `internal/importer`, `internal/relay`, `internal/catalog` e `queue` cuidam das demais operações. A documentação Astro fica em `docs`; `make docs.dev` exige suas dependências próprias.

`make e2e` roda as jornadas de `scripts/e2e.py` (administração, páginas públicas, permissões, isolamento, migração, encerramento) contra o binário real, com bancos novos e migrados criados em diretórios descartáveis, sem acessar o banco configurado em `.env`; o que cada jornada cobre está em `scripts/README.md`. Para validar uma cópia de banco de exemplo, use `python3 scripts/validate_multiuser.py --binary ./tunetrail --example-db /caminho/koito.db`; o original é aberto somente para leitura. A prévia fictícia isolada é iniciada por `make ui.preview`.

## Arquivos da interface

Os HTMLs estão em `assets/templates/public/` e `assets/templates/admin/`; CSS, JS, ícones e temas ficam em subpastas de `assets/`. O Makefile exporta `TUNETRAIL_WEB_DIR` e `TUNETRAIL_LANGS_DIR` para o checkout. Para `go run`, `go test` ou Delve fora do Makefile, informe esses caminhos explicitamente. A produção procura `assets/` ao lado do executável real, com symlinks resolvidos; um override explícito inválido falha sem fallback.

A carga cria um snapshot em memória por processo. Reinicie após editar arquivos; não é necessário recompilar para editar HTML/CSS. Não há hot reload nesta entrega. Público e administração usam conjuntos separados de templates. A base de renderização da ISSUE-065 roda em `make ui.check`/`make check` e mantém versão de fixture 0.10, relógio e dados fixos, independentemente da versão publicada.

`make package OUT=/tmp/tunetrail-new.tar.gz` cria uma distribuição local com binário, assets, catálogos e licenças. O destino precisa ser novo. A opção de embed permanece futura.

### Formatação dos templates

Os arquivos em `assets/templates/{public,admin,shared}/` usam indentação de dois espaços, blocos estruturais separados e atributos longos em várias linhas. Mantenha juntos os textos com elementos em linha quando uma quebra acrescentaria espaço visível; expressões Go longas podem ter os argumentos em linhas separadas sem alterar o texto produzido. Não quebre strings de tradução nem valores de atributos. Use `{{-` e `-}}` pontualmente para preservar espaços nas fronteiras de blocos condicionais.

Os blocos seguem três prefixos (ISSUE-067): `page/...` para documentos e seções de página (`page/live`, `page/preview`, `page/home`), `partial/...` para componentes reutilizados (`partial/head`, `partial/sidebar`, `partial/card`) e `admin/...` para a administração (`admin/home`, `admin/sidebar`). O `<head>` existe uma vez, em `assets/templates/shared/head.html`, montado nos dois conjuntos pelo manifesto e chamado com `{{template "partial/head" (head "live"|"preview"|"admin" título)}}`; o tipo escolhe as folhas e os scripts da página, sempre com nomes literais em `asset`.

`make ui.check` roda `scripts/check_web_style.py`: nenhuma linha acima de 160 caracteres em `assets/templates/` e `assets/css/`. As únicas exceções são linhas que não podem ser quebradas sem mudar a saída: um único atributo sem espaços no valor (uma URL longa) ou uma única string literal de tradução.

`make ui.reference.check` compara bytes exatos com a base atual em `backlog/evidencias/issue-067/reference/`, que inclui as URLs versionadas, o CSS derivado e os scripts de lifecycle. As bases históricas da ISSUE-065, da reformatação da ISSUE-066 e da ISSUE-106 permanecem intactas. A opção `--html-baseline` do comparador continua disponível para reproduzir a etapa histórica da 066, mas não é usada pelo gate atual. Mudanças de conteúdo exigem revisão; a comparação não ignora diferenças de whitespace para aprovar o gate.

Os alvos de teste do Makefile usam `-count=1`: mudanças nos arquivos externos lidos por `os.Root` não devem reaproveitar resultados anteriores do cache de testes Go.

### Verificação de cache e histórico

`make check` inclui os testes HTTP de cache, partida e condicionais e compara os 133 arquivos exportados com `backlog/evidencias/issue-067/reference/`. As bases históricas da 065, da formatação da 066 e da 106 continuam intactas. A base atual inclui URLs versionadas, CSS derivado e scripts de restauração revisados; a versão de apresentação das fixtures permanece fixa entre publicações.

Para verificar navegadores reais, execute `PLAYWRIGHT_MODULE=/caminho/playwright make web.cache.check`. São necessários Chrome e Firefox instalados, Node com WebSocket nativo e OpenSSL. `CHROME_PATH` e `TUNETRAIL_TEST_FIREFOX_PATH` selecionam os executáveis; os padrões macOS apontam para Applications. Cada execução cria dados fictícios, perfis temporários e um certificado HTTPS descartável; não altera a confiança do sistema nem o perfil pessoal do navegador. O Firefox usa WebDriver BiDi nativo. O roteiro verifica cache, documento/scroll/filtros restaurados, fallback de capas, idioma, seleção de perfis, fuso inválido, logout, expiração, DELETE e formulários administrativos. O Chrome também verifica restauração offline.

Para guardar o relatório: `python3 scripts/validate_web_cache.py --binary ./tunetrail --output /tmp/relatorio-novo --browser chrome` (ou `firefox`/`https`). O diretório deve ser novo. Não adicione `page.route` nem desative o cache nessas medições. O proxy local registra os bytes e as requisições reais dos assets, complementando Resource Timing quando uma navegação inicial é recarregada pelo navegador. Cookies e senhas não entram no relatório.
