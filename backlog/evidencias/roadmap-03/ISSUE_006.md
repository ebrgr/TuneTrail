# Issue 006 — núcleo de mídia implementado e testado

**Situação atual: issue encerrada por aceite do usuário em 05/11/2026.**
O restante preserva o histórico de implementação e verificação.

Registro de 4 de novembro de 2026. [Issue](<../../tasks/iss-006 - 006-—-Migrar-páginas-públicas-de-artistas-álbuns-e-faixas-para-Go.md>) ·
[Plano](<../../docs/roadmaps/doc-005 - Roadmap-3-—-artistas-álbuns-faixas-e-retrospectiva.md>).

**Estado: implementação funcional concluída; revisão visual adiada por orientação
do usuário para o final da issue 009.** Ativação pública e acesso transitório à
gestão permanecem na 009. A issue fica aberta até seu aceite visual.

## Entrega

- `internal/webui/media.go` resolve artista/álbum/faixa depois do slug, usando o
  contexto do perfil. IDs não inteiros/não positivos/overflow retornam 400;
  ausência ou falta de relação com o perfil retorna 404; falha interna retorna 500.
- `media_models.go` e `templates/media.html` apresentam capa, tipo, título, rank
  all_time, reproduções, minutos ouvidos e primeira escuta no fuso efetivo.
  Primeira escuta ausente não vira 1970. Compilações e múltiplos artistas mantêm
  seus metadados. MusicBrainz usa artist/release/recording conforme o tipo.
- Artista/álbum: oito faixas; histórico: onze registros filtrados; artista:
  até 99 álbuns associados/ouvidos pelo perfil. Títulos e continuação levam aos
  rankings/histórico completos com slug e filtro corretos. Mais de 99 álbuns não
  são apresentados como fim da coleção.
- Tocando agora vem do cache do perfil e só aparece quando combina com a entidade.
  Erros opcionais têm mensagem própria; não substituem o cabeçalho por zero.
- Consultas SQLite dos detalhes agora propagam falhas de contagem, duração e rank.
  Rank ausente para item ainda não ouvido em uma leitura interna de catálogo
  continua sendo zero; isso não autoriza a página pública sem relação pessoal.
- Handlers JSON preservam os campos e sucesso existentes; item inexistente segue
  404. Falha interna passa a 500 sem detalhe SQL; IDs zero/negativos passam a 400.
  A correção está registrada no changelog.

`IsPage`, a ponte React e as rotas da aplicação não foram ampliados nesta issue.
Os novos handlers são exercitados diretamente pelos testes. Em `KOITO_WEB_UI=go`,
URLs públicas de mídia **continuam em React**. Gráficos dependem da 007; rewind
da 008; gestão e ativação do conjunto da 009. Não foram criados controles fictícios
para essas próximas entregas.

## Verificação executada

`make check ui.check` passou: vet, suíte Go completa, typecheck e oito testes do
cliente, três testes de tema e sintaxe do JavaScript. Binário Go foi compilado.
Como nenhum código/dependência React mudou nesta issue, foi usado seu build local
existente para os testes HTTP.

`go test -race ./internal/webui ./internal/db/sqlite ./internal/webquery -timeout 90s`
passou, sem condição de corrida detectada nos pacotes exercitados.

Novos casos em `internal/webui/media_test.go` cobrem:

- Os três cabeçalhos comparados ao JSON da API, para dois perfis com números
  diferentes do mesmo catálogo; primeira escuta e HTML escapado.
- Mídia sem escutas no perfil, perfil vazio/ausente, IDs e filtros contraditórios.
- Múltiplos artistas, Various Artists, capas do fallback existente e tipos MBID.
- Falhas essenciais versus opcionais, sem expor mensagens privadas do store.
- Tocando agora compatível/incompatível e cache independente por perfil.
- Oito faixas, onze escutas, 99 álbuns e continuação após esse limite.
- Empates de rank, ausência de primeira escuta e dispatcher ainda em React.

`internal/db/sqlite/media_read_test.go` cobre falhas reais em banco descartável:
metadados válidos com tabela de escutas indisponível e duração indisponível na
agregação. Também preserva leituras internas de catálogo sem escutas/rank.

Validações com o binário real:

- `validate_webui.py`: páginas já migradas, paridade HTML/API, histórico página 2,
  filtros inválidos, autorização/origem e integridade; mídia continua na SPA.
- `validate_multiuser.py`: PASS em banco novo e migrado de v2, incluindo ingestão
  concorrente, administração, chaves, importação/exportação, revogação e reinício.

[validation-006.json](validation-006.json) reúne os resultados sem credenciais.
Os bancos são descartáveis em `.local/issue-006-*`, com esquema final **v3**.
Não houve migration nem uso do banco da instalação em execução. Os processos
transitórios das validações encerraram ao concluir; o serviço em 4110 solicitado
anteriormente não foi reiniciado.

## Revisão visual programada

Por orientação expressa do usuário, **não foi feita inspeção de layout ou captura
nova nesta issue**. O CSS e os templates estão implementados, mas ainda não têm
aceite visual. Ao final da 009, conferir juntos os cabeçalhos/listas da 006,
gráficos da 007, rewind da 008 e gestão/navegação da 009 em desktop/celular e gerar
os pares React/Go previstos no plano.

Nenhuma issue foi publicada; nenhum commit/push foi feito.
