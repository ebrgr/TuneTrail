# Issue 042 — integração da configuração e evidências

Validação realizada em 09/11/2026 com dados fictícios e bancos descartáveis. [Issue](<../../tasks/iss-042 - 042-—-Integração-e-aceite-da-configuração.md>) · [Contrato](<../../docs/design/doc-019 - Issue-038-—-contrato-e-plano-da-configuração-administrativa.md>) · [Capturas da interface](ISSUE_041.md).

## Matriz e jornadas

| Critério | Evidência |
| --- | --- |
| Arquivo, defaults, ambiente e `_FILE` | Testes `internal/cfg`: precedência, arquivo ausente/presente/inválido, `_FILE` vazio ou ilegível, validação de schema e das duas composições. |
| Persistência | Testes `internal/cfg/settings_store_test.go`: dez campos, comentários, concorrência, conflito, readonly, links e falhas de escrita antes/depois da publicação. |
| Salvar e reiniciar | `TestSettingsHTTPJourney` grava campos pela UI e reinicia a mesma instalação; verifica os dez valores ativos, override mantido, override removido em segundo reinício e ping Subsonic local. O mesmo teste garante que salvar não faz ping. |
| Permissões e resposta pública | Jornada real de login, sessão, CSRF, origem e negação de GET e dois POSTs com chave real de perfil, validada previamente no endpoint ListenBrainz. A resposta pública contém apenas `web_ui` e `default_theme`. |
| Fuso e Subsonic | `internal/webquery` cobre force/query/cookie/processo; `internal/images` cobre o ping limitado. A jornada HTTP testa rascunho sem salvar e servidor fictício. |
| Recuperação e reset | `TestTOMLStartupAndReset` inicia, executa reset, invalida TOML na mesma instalação, verifica que o banco não foi alterado, corrige offline e inicia novamente. |
| Relay, migração e importação | `TestProfileKeysCannotMoveAndRelayDoesNotReassign`; scripts `validate_multiuser.py` e `validate_admin_webui.py` exercitam bancos novos e migrados da v2, perfis, relay local, importação, exportação, reinício, esquema v3 e integridade SQLite. São jornadas complementares, não uma única jornada combinando todos os casos. |
| Interface | Capturas [desktop e celular](ISSUE_041.md), incluindo comparação com a administração existente, sem transbordamento nas larguras testadas; formulário testado sem JavaScript. Não é auditoria de acessibilidade completa. |

## Execução e implantação

- `CGO_ENABLED=0 go test -count=1 ./...`: passou; testes adicionais de integração após a alteração final também passaram.
- `scripts/validate_multiuser.py`: passou com bancos novo e migrado v2; integridade verificada.
- `scripts/validate_admin_webui.py`: passou com cinco formatos de importação, relay local e integridade.
- Dockerfile: imagem local construída com sucesso. Compose real com imagem local, volume e porta descartáveis: inicialização, reinício, `reset-password` com serviço parado e nova inicialização passaram. Contêiner e volume temporário removidos. A imagem local de teste permanece no Docker.
- `docker compose config` passou com variáveis fictícias. O exemplo [conf.example.toml](../../../deploy/conf.example.toml) não é carregado automaticamente.
- O JSON de lançamento do VS Code foi validado e o mesmo programa foi iniciado com Delve headless, `CGO_ENABLED=0`, `KOITO_WEB_UI=go` e diretório descartável; endpoint de saúde respondeu. **Não foi acionado F5 na interface gráfica nem lido o `.env` real**, portanto a integração visual do editor e particularidades do ambiente pessoal não foram verificadas.
- Não foram lidos nem modificados banco, arquivo `.env` ou configurações operacionais do usuário. SQLite continua em v3 e migrations não foram alteradas por esta issue.

## Revisão

[Advisor](../../../.agents/advisor.md), Sol esforço baixo, apontou inicialmente lacunas de reinício, recuperação e credencial real. Elas foram cobertas por testes; a revisão posterior aprovou. O contrato da resposta pública foi então reforçado para aceitar somente as duas propriedades esperadas.

A edição de valores exige reinício para aplicar configurações globais; variáveis de ambiente continuam tendo prioridade. Veja [documentação de configuração](../../../docs/src/content/docs/reference/configuration.md) para implantação e recuperação offline.
