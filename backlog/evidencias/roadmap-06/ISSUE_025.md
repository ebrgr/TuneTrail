# Issue 025 — migração, compatibilidade e dependências

Execução em 05/11/2026. [Issue](<../../tasks/issue-025 - 025-—-Validar-migração-compatibilidade-e-dependências-React.md>) ·
[Plano](<../../docs/roadmaps/doc-009 - Roadmap-6-—-comparação-visual-e-decisão-sobre-React.md>) ·
[Resultados HTTP e digests fictícios](validation-025.json).
Go é o padrão desde a 027; React só é executado com `KOITO_WEB_UI=react`.
Esta issue não remove interfaces, altera o padrão ou modifica as migrações.

## Migração e jornadas

`scripts/validate_migration_025.py` construiu **quatro bancos v2 independentes**
aplicando 001/002 a bases novas fictícias; não houve downgrade de uma base v3 nem
leitura do banco operacional. Em cada caso o startup real migrou a cópia para v3,
criou backup e preservou os digests das tabelas legadas antes de autenticação.
Cada backup foi restaurado para outra cópia e conferido como v2 com integridade e
mesmos digests. Uma das quatro bases tinha uma segunda escuta confirmada em WAL
ativo durante o startup; o backup também preservou esse dado. Não foi executado
um binário antigo contra a cópia restaurada.

A fixture migrada mínima contém um administrador, um perfil, uma chave, um artista,
um álbum, uma faixa e uma escuta legada; a variante WAL tem duas. É diferente da
fixture 022 e não comprova migração de todos os cenários de múltiplos usuários,
Unicode, DST ou históricos longos. Após a migração, as jornadas adicionam dados
fictícios em cópias separadas; as asserções de preservação são anteriores a elas.

- **Web Go e React**: navegador real acessou o perfil migrado, a rota legada
  `/listens` e seu destino `/default/listens`, fez login pela UI, editou o nome
  do perfil e releu o valor persistido; a escuta legada permaneceu. Em React o
  erro de perfil ausente foi visto após hidratação.
- **Admin Go**: `validate_admin_webui.validate` executou formulários e métodos
  HTTP, inclusive cinco formatos de importação, exportação, chaves, relay local,
  DST manual, exclusões, plano obsoleto/repetido e integridade.
- **Admin React/API**: outra base migrada executou `validate_journey` sobre a API,
  incluindo criação de perfis/chaves, ingestão, leituras, import/export,
  revogação e reinício. A edição de perfil por interface React foi exercitada na
  cópia web separada; a jornada administrativa completa é HTTP/API, não clique
  em todos os controles da SPA.
- Cada cópia manteve chave legada válida após migração e terminou com
  `integrity_check` e `foreign_key_check` aprovados. Os quatro backups foram
  conferidos antes das mutações. Referência dos digests e SHA-256 do binário de
  teste constam do JSON; o commit informado tinha alterações locais não commitadas.

## Rotas selecionadas após migração

Foram verificadas **50 combinações de modo, método e caminho**, não 50 rotas
independentes nem toda a matriz de métodos/atores. A tabela resume GET; HEAD foi
verificado com o mesmo status e corpo vazio. POST de criação de perfil por
visitante foi negado em ambos; status completo e tipos estão no JSON.

| Caminho | Go | React explícito | Evidência/limite |
| --- | --- | --- | --- |
| `/`, `/default`, `/admin` | 200 HTML Go | 200 shell HTML | Navegador confirmou navegação, login e edição nos dois modos |
| `/listens` | 302 para `/default/listens` | 200 shell, depois redirecionamento no cliente | Destino e histórico legados confirmados no navegador |
| `/favicon.svg`, `/public/yuu.jpg` | 200 SVG/JPEG | 200 SVG/JPEG | Favicon Go incorporado; `/public` vem de arquivos externos ao binário |
| `/apis/web/v1/profiles`, `/health` | 200 | 200 | JSON e health preservados; ListenBrainz com chave legada também validado |
| `/web-assets/ui.css` | 200 CSS | 200 HTML | Em React, caminho desconhecido recebe shell, não arquivo CSS |
| `/theme-helper` | 404 | 200 shell, **falha na hidratação** | Defeito reproduzido nesta execução histórica; [issue 028 resolvida](<../../tasks/issue-028 - 028-—-Corrigir-falha-de-contexto-no-theme-helper-React.md>) |
| Asset ou página inexistente | 404 | 200 shell | O shell 200 não garante recurso existente; perfil React ausente apresentou erro hidratado |

Na 025, `/theme-helper` React gerou `This screen requires a listening profile`
antes de mostrar o editor. O Advisor atribuiu por leitura os componentes fora de
`ProfileProvider` à rota global; a falha também foi reproduzida em fixture v3.
A aba de teste foi fechada ao detectar o erro para interromper o ciclo de erros.
O defeito foi posteriormente corrigido na [028](ISSUE_028.md); esta descrição
preserva o resultado da execução histórica da 025.

## Inventário para decisão futura

| Área | Estado observado | Destino antes de remover React |
| --- | --- | --- |
| Configuração e rotas | `cfg` escolhe Go por padrão; modo React explícito. Go despacha páginas/ações conhecidas e 404; React usa SPA fallback | Preservar URLs documentadas e definir explicitamente caminhos que hoje pertencem à SPA, inclusive 028 |
| HTML, CSS, JS e favicon Go | `internal/webui/preview.go` embute templates, `static` e paletas; `internal/adminui/admin.go` embute templates administrativos | Verificar comportamento do binário distribuído; favicon Go já embutido |
| `/public` e imagens | `engine/routes.go` usa `client/public` relativo ao diretório de trabalho; `/image` usa cache | Inventariar/copiar ou incorporar arquivos necessários; não pressupor binário autossuficiente |
| SPA e cliente | `client/build/client` é servido apenas em modo React, relativo ao diretório de trabalho; React Router `ssr:false` | Preservar build até decisão de remoção; definir tratamento de erros e assets inexistentes |
| Docker | Estágio Node 24 compila frontend; imagem final copia SPA e `client/public`, usa libvips/tzdata | Planejar e validar pacote Go-only separadamente; não alterar Docker nesta issue |
| Makefile e CI | `setup`, `doctor`, `check`, `build` e `multiuser.check` ainda usam ferramentas ou build frontend; `ui.check` usa Node nos testes JS | Separar requisitos e checks em proposta futura; manter os existentes enquanto React existir |
| Preview e scripts | `cmd/ui-preview` pode usar referência React; capturadores Playwright e validadores dependem de ambas as interfaces | Atualizar/remover referências somente após aprovação; capturas antigas continuam evidência histórica |
| Fontes | Templates Go e cliente pedem Google Fonts | Avaliar fontes locais/fallback e licenças; ainda não há prova de paridade tipográfica offline |
| Documentação/VS Code | `KOITO_WEB_UI`, builds, modos, rotas e depuração documentados; perfil React explícito continua disponível | Revisar instruções e distribuição no momento de eventual retirada |

## Rollback e pendências anteriores

Trocar `KOITO_WEB_UI=react` e reiniciar é reversão **de interface**, com o mesmo
esquema v3; não desfaz dados nem exige reverter a migration. Reverter para uma
versão que dependa do esquema v2 exige planejar restauração de backup compatível:
a migração 003 bloqueia deliberadamente `Down`, e restaurar o snapshot não inclui
as gravações posteriores. Testamos restauração em cópias fictícias, não um
rollback de produção nem compatibilidade com binário antigo.

| Tema das issues 018–021 | Situação/destino |
| --- | --- |
| Datas IANA inteiramente removidas, offsets manuais históricos em segundos | Limites documentados na 018; não reexecutados sobre v2 migrada; encaminhar à 026 para decisão de escopo |
| Intervalos de interesse | Oráculo independente da 023 conferiu API e tabela Go em v3; não repetido para todos os intervalos em base migrada nem extração numérica do gráfico React |
| Falha de disco/limpeza, reinício durante importação e interleavings | Continuam lacunas da 020; backup com WAL confirmado testado aqui não prova falha real de disco ou todas as ordens de concorrência |
| Import parcial, export dinâmico e relay best-effort | Contratos preservados e limites registrados na 020; não são garantia de rollback integral ou entrega durável |
| Compressão bimg, callbacks e tarefas que ignoram cancelamento | Limites da 020 continuam; sem alegar encerramento cooperativo universal |
| Isolamento público e contexto interno global | Cobertura 019/021 e jornadas desta issue são finitas; não certificam todos os adaptadores |
| Fonts, responsividade e acessibilidade | Evidências 023/024 e limitações pendentes de avaliação consolidada na 026 |
| `/theme-helper` React | Falha reproduzida nesta issue; correção e testes posteriores em [028](<../../tasks/issue-028 - 028-—-Corrigir-falha-de-contexto-no-theme-helper-React.md>) |

## Execução e revisão

`PLAYWRIGHT_MODULE=<instalação local>/playwright python3 scripts/validate_migration_025.py
--binary <binário> --report <arquivo-novo>` passou com quatro migrações, backups,
restaurações, 50 combinações HTTP, navegador e integridade. Go 1.26.3, Node
v24.15.0, Chrome/Playwright e SQLite disponíveis localmente; servidores simulados
em loopback e rede externa bloqueada no navegador. Não foram necessárias capturas
para comprovar o comportamento de migração. [Advisor](../../../.agents/advisor.md),
Sol esforço médio: **aprovado com ressalvas**, com o defeito 028 e limites acima.
