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

No VS Code, instale a extensão Go e use **Koito: Desenvolvimento Go (localhost:4110)** com F5. O Delve inicia a aplicação e abre o navegador quando ela estiver pronta. Pare uma instância anterior se a porta estiver ocupada. `make api.debug` inicia o Delve no terminal. Reinicie o servidor após alterar código, templates ou assets incorporados.

## Estrutura e validação

`cmd/api` contém a entrada; `engine` concentra rotas e handlers; `internal/webui` e `internal/adminui` implementam as páginas públicas e administrativas; `internal/db/sqlite` e `db/migrations_sqlite` contêm persistência e migrações. `internal/importer`, `internal/relay`, `internal/catalog` e `queue` cuidam das demais operações. A documentação Astro fica em `docs`; `make docs.dev` exige suas dependências próprias.

`make e2e` roda as jornadas de `scripts/e2e.py` (administração, páginas públicas, permissões, isolamento, migração, encerramento) contra o binário real, com bancos novos e migrados criados em diretórios descartáveis, sem acessar o banco configurado em `.env`; o que cada jornada cobre está em `scripts/README.md`. Para validar uma cópia de banco de exemplo, use `python3 scripts/validate_multiuser.py --binary ./tunetrail --example-db /caminho/koito.db`; o original é aberto somente para leitura. A prévia fictícia isolada é iniciada por `make ui.preview`.
