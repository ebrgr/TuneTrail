# ISSUE-127 — Estudo: Subsonic/Navidrome na configuração do servidor

Leitura estática da `main` em `efb2f8a` (v0.68). Pedido do usuário em 23/11/2026: decidir o que entra na UI e o que não entra, e avaliar um botão que abre um popup com usuário e senha do Navidrome para montar a string de parâmetros (`u`, `t`, `s`). O popup mostra um aviso de que a senha nunca é gravada nem enviada em aberto. Este estudo não altera código.

## 1. Estado atual

| Ponto | Onde | Hoje |
| --- | --- | --- |
| Campos | `internal/adminui/settings.go:59-60` | `Subsonic URL` e `Subsonic parameters`, ambos `text`, sem ajuda para montar a string. |
| Validação | `internal/cfg/toml.go:208-233` | URL e parâmetros juntos ou vazios. Exige `u` e (`p`) ou (`t` + `s`). Rejeita `f`, `v`, `c`, `query`, `id` e contadores. Preserva extensões. |
| Teste | `internal/images/subsonic_ping.go`, `settings.go:301-311` | `POST /admin/settings/subsonic/test` usa o rascunho exibido, sem salvar. Chama `/rest/ping.view` e confere `status=ok`. |
| Guia ao usuário | `docs/src/content/docs/quickstart/navidrome.md` | Manda abrir o DevTools do navegador, achar um `getCoverArt` e copiar `u=…&t=…&s=…`. É o passo que o popup substitui. |
| Exibição | `assets/templates/admin/settings.html:97-114` | Cada campo mostra Salvo, Efetivo e Em execução em texto, **inclusive `t` e `s`**. |
| JS | `shared/head.html:10-22` | Scripts próprios com `defer`. Já existe `<dialog>` modal em `media-dialog.js`. |

A página funciona sem JavaScript e precisa continuar assim (AC 3 da issue).

## 2. O que entra na UI e o que não entra

**Entra (cartão Subsonic, junto da 127):**

1. Campos URL e Parâmetros, como hoje, com origem do valor e aviso de reinício no cartão (ACs 1 e 2).
2. Botão **Gerar parâmetros** ao lado do campo Parâmetros. Só aparece com JavaScript e `<dialog>`. Fica desabilitado, com a explicação padrão, quando `TUNETRAIL_SUBSONIC_PARAMS` vem do ambiente (campo bloqueado).
3. Popup com usuário e senha, aviso fixo, botões **Gerar** e **Cancelar**. O resultado preenche só o campo Parâmetros e devolve o foco a ele. Não salva nada.
4. **Testar sem salvar** mantido na barra de ações (rodapé alcançável).
5. Máscara de `t` e `s` nas linhas Salvo/Efetivo/Em execução (ver §5, decisão 2).

**Não entra:**

| Item | Motivo |
| --- | --- |
| Senha em texto (`p=`, `p=enc:hex`) | O popup só gera `t`+`s`. A validação continua aceitando `p` por arquivo/ambiente para quem já usa, mas a UI não incentiva. |
| Edição separada de `f`, `v`, `c` | São reservados e rejeitados pela validação; o cliente fixa `v=1.13.0` e `c=tunetrail`. |
| Extensões de parâmetro avulsas | Servem a quem edita o texto à mão; sem campo próprio. |
| Gravar usuário ou senha em qualquer lugar (SQLite, TOML, `localStorage`, cookie, log) | Contraria o aviso do popup. Só o resultado `u`/`t`/`s`, que o usuário decide salvar, persiste. |
| Envio da senha ao servidor TuneTrail para ele calcular o token | Ver §3: o cálculo é local. |
| Seletor de servidor, descoberta automática, “login com Navidrome” | Fora do escopo; a URL continua manual. |
| Chamada do navegador ao Navidrome (testar de dentro do popup) | Exigiria CORS no Navidrome. O teste continua pelo servidor, com o token já gerado. |

## 3. Desenho do popup e segurança

**Cálculo no navegador.** O token Subsonic é `t = md5(senha + sal)`. O JS gera `s` com `crypto.getRandomValues` (16 hex), calcula `t` localmente e escreve `u=<usuário>&t=<md5>&s=<sal>` no campo. A senha nunca sai do navegador, nem em POST, nem em query, nem em fetch. Isso torna verdadeiro o aviso «a senha nunca é gravada nem enviada».

- WebCrypto não oferece MD5; é preciso uma implementação própria, pequena e inline em `assets/js/` (sem dependência externa, pois não há bundler). Cobrir com vetores RFC 1321 num teste.
- Senha em `<input type="password" autocomplete="off">` **fora** do `<form>` de configurações, para que um Enter ou o envio sem JS nunca a transmita. O popup usa `<form method="dialog">` ou nenhum `<form>`; nenhum `name` em campo de senha.
- Ao gerar, cancelar, fechar ou perder a página, apagar o valor do campo (`input.value = ''`) e o `<dialog>` tem `autocomplete` desligado.
- Usuário com `&`, `=` ou espaço: codificar com `encodeURIComponent`, ou `URLSearchParams`, para casar com `url.ParseQuery` do Go.
- Sem JS: o botão não existe e o campo continua editável como hoje, com o texto do quickstart.

**Texto do aviso (rascunho, a revisar com o usuário):**

> A senha é usada só neste navegador para calcular o token e nunca é gravada nem enviada ao TuneTrail. O que você salva é o token (`t`) e o sal (`s`): quem os tiver consegue acessar o Navidrome como esse usuário até a senha mudar. Use um usuário dedicado, de preferência sem privilégio de administrador.

Dizer isso é parte da honestidade do aviso: `t`+`s` equivalem a uma credencial. Trocar a senha no Navidrome invalida o token.

**Acessibilidade:** `<dialog>` modal com título, foco inicial no usuário, Esc fecha, foco volta ao botão, alvos de 44 px, rótulos acima dos campos, aviso ligado por `aria-describedby`. Mensagem de erro (campo vazio) dentro do popup, sem enviar nada.

## 4. Mapa de implementação

| Peça | Arquivo | Observação |
| --- | --- | --- |
| Cartão e botão | `assets/templates/admin/settings.html` | Ao lado do campo `SUBSONIC_PARAMS`; `<dialog id="subsonic-credentials">` renderizado pelo servidor. |
| i18n | `assets/templates/…` + catálogo de textos | Chaves novas para título, rótulos, aviso e erro, em pt-BR e en. |
| Lógica | `assets/js/subsonic-credentials.js` (novo) ou `admin.js` | MD5, sal, preenchimento do campo, limpeza. Registrar em `shared/head.html` só na página de configurações. |
| CSS | `assets/css/admin/` | Padrão do cartão (bg-secondary, borda bg-tertiary, raio 8 px). |
| Máscara | `internal/adminui/settings.go` | Função de exibição só para `SUBSONIC_PARAMS`; o `value` do input segue completo. |
| Testes | `internal/adminui/…_test.go` | Botão presente; `<dialog>` sem `name` no campo de senha; botão ausente com campo bloqueado; máscara. MD5 por vetores em teste de JS (`node`) ou, se não houver infraestrutura, por teste Go que executa o arquivo. Ver §6. |
| Docs e changelog | `docs/…/navidrome.md`, `CHANGELOG.md` | Trocar o passo do DevTools pelo popup. Entrada em inglês, em `[Unreleased]`. |

Sem mudança de schema, de rota nem de contrato: o `POST /admin/settings/save` e o teste recebem a mesma string.

## 5. Decisões do usuário (23/11/2026)

1. **Popup:** `<dialog>` com cálculo no navegador. Confirmado como viável (MD5 próprio, testado contra os vetores da RFC 1321).
2. **Exibição de estado:** mantida como está, sem mascarar `t` e `s`.
3. **Aviso:** mantida a frase de que `t`+`s` equivalem a uma credencial.
4. **Escopo:** novo AC na 127 («Popup gera `u`/`t`/`s` sem enviar nem gravar a senha»).
5. **Teste de conexão:** botão **Testar conexão** dentro do próprio `<dialog>`, além do botão da barra de ações.

## 6. Plano de verificação

1. Vetores MD5 (RFC 1321) e um caso `md5(senha+sal)` conhecido, comparado ao `md5` do sistema.
2. `go test ./internal/adminui/... ./internal/cfg/...` e `make check`.
3. Servidor de teste: `https://musica.bergerlab.com.br`, Navidrome 0.64.2 (OpenSubsonic), alcançável. Sem credenciais, `ping.view` responde `missing parameter: 'u'`, como esperado.
   - Com credenciais: gerar `t`/`s` pelo popup, usar **Testar sem salvar** na worktree e conferir `status=ok`.
   - Credenciais fora de qualquer arquivo da worktree: sem `conf.toml` com o token, sem captura de tela com o popup preenchido, sem fixture. Rodar a instância de teste com `TUNETRAIL_DATA_DIR` em diretório temporário e apagá-lo no fim.
   - Recomendação: usar um usuário Navidrome descartável, só leitura, e removê-lo depois. As observações do `ai-memory` registram eventos de ferramenta; uma senha digitada na conversa ou num comando pode ficar no registro da sessão.
4. Capturas 390/768/1440 em Yuu e Pearl, com o popup aberto e vazio (AC 4).

## 7. Variáveis de ambiente: auditoria completa (23/11/2026)

Fontes: `internal/cfg/cfg.go` (34 variáveis `KOITO_*`/`TUNETRAIL_*`), `RELAY_ALLOW_HTTP` e `RELAY_ALLOW_PRIVATE_NETWORK` (`settings_edit.go`), `TUNETRAIL_WEB_DIR` e `TUNETRAIL_LANGS_DIR` (lidas fora do pacote `cfg`), mais `TZ` do processo. Cada nome aceita `KOITO_`, `TUNETRAIL_` e o sufixo `_FILE`. Uso real conferido por chamadores dos getters. Neste ambiente não há nenhuma variável `TUNETRAIL_*` definida.

### 7.1 Página × `conf.toml`: 12 de 12

Cada campo da página tem uma chave no `conf.toml`, e o contrário também vale. Provado por teste (`TestEveryManagedSettingHasATOMLKey` em `internal/cfg` e `TestSettingsPageMatchesManagedSettings` em `internal/adminui`): o leitor TOML devolve exatamente as 12 configurações gerenciadas e a página mostra exatamente essas 12.

| Cartão | Campo | Chave no `conf.toml` |
| --- | --- | --- |
| Geral e log | Nível de log | `logging.level` |
| Horário | Fuso da aplicação | `time.force_tz` |
| MusicBrainz | URL | `musicbrainz.url` |
| MusicBrainz | Limite de requisições | `musicbrainz.rate_limit` |
| MusicBrainz | Desativar | `musicbrainz.disabled` |
| Imagens | Desativar Deezer | `images.disable_deezer` |
| Imagens | Desativar Cover Art Archive | `images.disable_cover_art_archive` |
| Imagens | Chave Last.fm | `images.lastfm_api_key` |
| Subsonic | URL | `subsonic.url` |
| Subsonic | Parâmetros | `subsonic.params` |
| Rede e relay | Relay por HTTP | `relay.allow_http` |
| Rede e relay | Relay em rede privada | `relay.allow_private_network` |

Lacuna de documentação: `configuration.md` documenta as chaves TOML do relay, mas não os nomes de variável `RELAY_ALLOW_HTTP` e `RELAY_ALLOW_PRIVATE_NETWORK`.

### 7.2 O que retirar da página: nada

Os 12 campos têm efeito real no código (todos têm leitor em tempo de execução) e nenhum é duplicado. Observação, não remoção: a chave Last.fm aparece em texto, como o Subsonic (decisão do usuário: sem máscara).

### 7.3 Variáveis mortas ou legadas (fora da página, candidatas a limpeza de código e documentação)

Isto corrige a tabela anterior do estudo, que as classificava como «inicialização» e «outras».

| Variável | Situação encontrada | Sugestão |
| --- | --- | --- |
| `ALLOWED_HOSTS` | O motor só registra «no longer used and can be removed» (`engine/engine.go:225`). Nem consta na referência. | Remover a leitura. |
| `ENABLE_FULL_IMAGE_CACHE` | Só é citada no mapa de nomes (`env_alias.go`); nenhum código a lê. Também ausente da referência. | Remover. |
| `LOGIN_GATE` | Lida em `cfg.go:293`, mas `LoginGate()` não tem nenhum chamador. A referência promete «não mostra estatísticas sem login», o que o código não faz hoje (a proteção é por perfil). | Corrigir a documentação ou remover; decisão do usuário. |
| `ENABLE_LBZ_RELAY`, `LBZ_RELAY_URL`, `LBZ_RELAY_TOKEN` | Legado: associação única a um perfil; o relay é por perfil. | Manter até a 164; não vão para a página. |

Registrado na **ISSUE-197** (limpeza de código e referência). Não pertence à 127.

### 7.4 O que acrescentar à página

| Variável | Proposta | Por quê |
| --- | --- | --- |
| `ENABLE_STRUCTURED_LOGGING` (**ISSUE-198**) | **Acrescentar** ao cartão Geral e log, como `logging.structured` (bool). | Segura, sem segredo, nada de inicialização crítica; segue o padrão dos 12 (reinício para valer). Exige nova chave em `toml.go` e em `settings_edit.go`. |
| `ARTIST_SEPARATORS_REGEX` (**ISSUE-199**, estudo) | **Depois**, com validação da expressão e um cartão «Catálogo». | Altera como novos artistas são associados; uma regex inválida hoje impede a subida, então precisa ser validada ao salvar. |
| `DEFAULT_THEME` | Não aqui. | Pertence à ISSUE-153, que persiste as opções do painel de tema. |
| `FETCH_IMAGES_DURING_IMPORT`, `THROTTLE_IMPORTS_MS`, `IMPORT_BEFORE_UNIX`, `IMPORT_AFTER_UNIX`, `SKIP_IMPORT` | Não. | Valem na carga de importação na subida, não são configuração do servidor em funcionamento. |
| `DISABLE_RATE_LIMIT`, `CORS_ALLOWED_ORIGINS`, `LOGIN_GATE` | Não. | Mudam a proteção do próprio acesso; um erro se paga com a perda de controle. Ficam em variável de ambiente. |
| `BIND_ADDR`, `LISTEN_PORT`, `CONFIG_DIR`, `WEB_UI`, `SQLITE_ENABLED`, `DATABASE_URL`, `TUNETRAIL_WEB_DIR`, `TUNETRAIL_LANGS_DIR` | Não. | Definem como o processo sobe ou onde estão os arquivos. |
| `DEFAULT_USERNAME`, `DEFAULT_PASSWORD`, `LBZ_RELAY_TOKEN` | Não, e jamais exibir. | Segredos. A página mostra o valor salvo em texto. |

Sugestão adicional, registrada na **ISSUE-200**: mostrar a chave do `conf.toml` (por exemplo `logging.level`) dentro dos «detalhes» de cada campo, já que `cfg.TOMLKey` existe.

## 8. Verificação feita (23/11/2026)

- Node: 4 testes (`internal/webui/subsonic-credentials.test.mjs`): vetores RFC 1321, fronteiras de preenchimento e UTF-8 contra o `crypto` do Node, composição de `u`/`t`/`s` e ausência da senha na saída.
- Go: `internal/adminui`, `internal/webassets`, `internal/webui`, `internal/i18n`, `internal/cfg` verdes (com `TUNETRAIL_WEB_DIR=assets`).
- Instância local isolada (porta 4199, diretório temporário apagado depois) contra `https://musica.bergerlab.com.br` (Navidrome 0.64.2): **Testar conexão** no popup retorna sucesso com usuário descartável; senha errada e campos vazios mostram mensagem dentro do popup; a senha é apagada ao gerar, cancelar ou fechar, e permanece depois de **Testar conexão** para que **Gerar parâmetros** possa vir em seguida; **Gerar parâmetros** preenche só `u`, `t` e `s`, fecha o popup e devolve o foco ao campo; o botão **Testar Subsonic sem salvar** da página funciona em seguida com os tickets renovados.
- Capturas do popup vazio: `popup-390-yuu.png` e `popup-1440-pearl.png`. As capturas completas 390/768/1440 da página em Yuu e Pearl continuam pendentes para o AC 4.
- Nenhuma credencial em arquivo da worktree, em captura ou em fixture.
