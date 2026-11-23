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
2. Botão **Gerar parâmetros…** ao lado do campo Parâmetros. Só aparece com JavaScript e `<dialog>`. Fica desabilitado, com a explicação padrão, quando `TUNETRAIL_SUBSONIC_PARAMS` vem do ambiente (campo bloqueado).
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

## 7. Variáveis de ambiente: o que fica na tela e o que sai

Auditoria de `internal/cfg/cfg.go` (36 variáveis), `managedDefaults` em `internal/cfg/toml.go:61-65` e `settingsFields` em `internal/adminui/settings.go:51-64`. Neste ambiente de desenvolvimento nenhuma variável `TUNETRAIL_*` ou `KOITO_*` está definida.

**Na tela hoje (12, as únicas gerenciáveis pelo TOML e pela UI):** log level, MusicBrainz (URL, taxa, desligar), Deezer, Cover Art Archive, Last.fm, Subsonic (URL, parâmetros), fuso, relay HTTP e relay em rede privada. Proposta de cartões da 127: **Geral e log** (nível, fuso), **MusicBrainz** (URL, taxa, desligar), **Imagens** (Deezer, CAA, Last.fm), **Subsonic** (URL, parâmetros, popup), **Rede** (relay HTTP e privada).

**Fora da tela, e devem continuar fora:**

| Grupo | Variáveis | Motivo |
| --- | --- | --- |
| Segredos e acesso | `DEFAULT_USERNAME`, `DEFAULT_PASSWORD`, `DATABASE_URL`, `LBZ_RELAY_TOKEN` | Exibir valor seria vazamento; a tela mostra Salvo/Efetivo em texto. |
| Inicialização | `BIND_ADDR`, `LISTEN_PORT`, `CONFIG_DIR`, `WEB_UI`, `SQLITE_ENABLED`, `ALLOWED_HOSTS`, `CORS_ALLOWED_ORIGINS`, `LOGIN_GATE`, `DISABLE_RATE_LIMIT` | Mudam como o processo sobe ou a proteção da própria tela de administração; um erro trancaria o acesso. |
| Importação | `SKIP_IMPORT`, `THROTTLE_IMPORTS_MS`, `IMPORT_BEFORE_UNIX`, `IMPORT_AFTER_UNIX`, `FETCH_IMAGES_DURING_IMPORT` | Valem na primeira carga, não são configuração de servidor. |
| Legado | `ENABLE_LBZ_RELAY`, `LBZ_RELAY_URL` | O relay é por perfil; a tela já avisa que o legado fica fora. |
| Outras | `ENABLE_STRUCTURED_LOGGING`, `ENABLE_FULL_IMAGE_CACHE`, `ARTIST_SEPARATORS_REGEX`, `DEFAULT_THEME` | Candidatas futuras. `DEFAULT_THEME` já pertence à 153. |

**Efeito sobre a UI do popup:** a variável `KOITO_SUBSONIC_PARAMS` (ou `TUNETRAIL_SUBSONIC_PARAMS`, ou `_FILE`) bloqueia o campo; nesse caso o botão do popup não é renderizado. Coberto por teste Go.

**Achado para decisão futura:** a chave da Last.fm também aparece em texto na entrada e em Salvo/Efetivo/Em execução. O usuário manteve a exibição sem máscara para o Subsonic; a mesma decisão vale por ora para a Last.fm, mas fica registrada.

## 8. Verificação feita (23/11/2026)

- Node: 4 testes (`internal/webui/subsonic-credentials.test.mjs`): vetores RFC 1321, fronteiras de preenchimento e UTF-8 contra o `crypto` do Node, composição de `u`/`t`/`s` e ausência da senha na saída.
- Go: `internal/adminui`, `internal/webassets`, `internal/webui`, `internal/i18n`, `internal/cfg` verdes (com `TUNETRAIL_WEB_DIR=assets`).
- Instância local isolada (porta 4199, diretório temporário apagado depois) contra `https://musica.bergerlab.com.br` (Navidrome 0.64.2): **Testar conexão** no popup retorna sucesso com usuário descartável; senha errada e campos vazios mostram mensagem dentro do popup; a senha é apagada do campo após cada ação; **Gerar parâmetros** preenche só `u`, `t` e `s`, fecha o popup e devolve o foco ao campo; o botão **Testar Subsonic sem salvar** da página funciona em seguida com os tickets renovados.
- Capturas do popup vazio: `popup-390-yuu.png` e `popup-1440-pearl.png`. As capturas completas 390/768/1440 da página em Yuu e Pearl continuam pendentes para o AC 4.
- Nenhuma credencial em arquivo da worktree, em captura ou em fixture.
