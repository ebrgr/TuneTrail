# Issue 041 — administração da configuração

Implementação em 09/11/2026. [Issue](<../../tasks/iss-041 - 041-—-Edição-de-configuração-pelo-master.md>) ·
[Contrato](<../../docs/design/doc-019 - Issue-038-—-contrato-e-plano-da-configuração-administrativa.md>) ·
[Persistência 040](ISSUE_040.md).

## Entrega

A rota `/admin/settings` usa a sessão administrativa existente e disponibiliza os
dez campos globais. O formulário mostra os valores salvos, efetivos após reinício,
ativos no processo, suas origens e os overrides do ambiente. Valores Last.fm e
Subsonic aparecem como texto, conforme solicitado. O fuso local do processo é
mostrado com zona e offset; fuso forçado vazio mantém seleção automática.

O salvamento usa revisão e persistência da 040. Conflitos mantêm o rascunho e a
revisão anterior com HTTP 409; uma publicação cuja durabilidade não pôde ser
confirmada é informada como publicada, com a nova revisão. Erros de validação
mantêm os valores e recebem novo ticket. Se a leitura do estado falhar, a tela
não inventa valores salvos/ativos. As alterações globais efetivas exigem reinício.

O botão **Test Subsonic without saving** valida o par URL/params e testa apenas
o rascunho com o helper limitado da 039; não salva nem altera o cliente ativo.
Salvar não faz acesso remoto. A tela informa que overrides podem alterar o valor
efetivo. Não há sincronização de escutas Subsonic.

O relay continua no SQLite por perfil. Há link para sua gestão, e o token aparece
em campo de texto preenchido. URL e token precisam estar presentes para salvar
ou testar; **Remove relay** desativa a associação. Valores vazios não são mais
interpretados como pedido implícito para reutilizar o token anterior.

## Comparação visual

Capturas de instância descartável com dados fictícios, tema Yuu e fontes externas
bloqueadas. A prévia foi encerrada após os testes.

| Largura | Administração existente | Configuração nova |
| --- | --- | --- |
| Desktop 1440×1000 | [Administração](issue-041/admin-desktop.png) | [Configuração](issue-041/settings-desktop.png) |
| Celular 390×844 | [Administração](issue-041/admin-mobile.png) | [Configuração](issue-041/settings-mobile.png) |

A tela reutiliza a barra lateral, cores e componentes administrativos. Nas duas
larguras o documento não transbordou horizontalmente; o primeiro foco pelo teclado
foi o link para o conteúdo. Um navegador com JavaScript desligado conseguiu abrir,
editar e salvar a configuração. [Dados das capturas](issue-041/results.json).

Esta comparação não equivale a auditoria completa de acessibilidade ou de todos
os temas, navegadores e tamanhos. A integração e o aceite ampliados ficam na 042.

## Validação

- Suíte Go completa sem CGO e `go vet ./...`: passaram; testes JavaScript de tema
  e verificação sintática dos scripts da interface também passaram.
- `go test -race` em adminui, adminweb e cmd/api: passou.
- Jornada HTTP real (`TestSettingsHTTPJourney`): login/retorno, negação de chave
  de ingestão em GET/POST, CSRF/origem, limites/duplicatas, valores visíveis,
  persistência, override, reinício pendente, erro de fuso com rascunho, HEAD,
  conflito entre abas, validação/teste Subsonic e ausência de teste remoto no save.
- Testes dirigidos cobrem erro de durabilidade após publicação, estado salvo
  indisponível, opções inválidas preservadas e token de relay visível.
- `validate_multiuser.py` e `validate_admin_webui.py`: passaram com bancos novos
  e migrados e dados fictícios; integridade SQLite preservada. Builds Linux amd64
  e arm64 sem CGO passaram; não foram executados nesta verificação.
- [Advisor](../../../.agents/advisor.md), Sol esforço baixo, revisou plano e patch.
  As ressalvas sobre estado indisponível e seleção inválida foram corrigidas e
  testadas; parecer final aprovado. A apresentação do relay também foi revisada.

## Reprodução das capturas

Usar somente uma instância com banco **descartável**: o teste sem JavaScript
salva o valor fictício `preview-value` no campo Last.fm. Dependências de captura
são ferramentas externas de desenvolvimento, não dependências do aplicativo.

```sh
PLAYWRIGHT_MODULE=/caminho/para/playwright \
TUNETRAIL_PREVIEW_ORIGIN=http://127.0.0.1:14141 \
TUNETRAIL_PREVIEW_USER=admin \
TUNETRAIL_PREVIEW_PASSWORD=senha-da-fixture \
node scripts/capture_settings_041.cjs backlog/evidencias/configuracao/issue-041
```

Chrome pode ser escolhido por `CHROME_PATH`. O script bloqueia requests para
origens externas e salva as métricas em `results.json`.
