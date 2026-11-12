# Issue 019 — isolamento, ingestão e cache

Implementada e aceita pelo usuário em 05/11/2026. [Issue](<../../tasks/iss-019 - 019-—-Validar-isolamento-entre-perfis-ingestão-e-cache.md>) · [Matriz inicial](ISSUE_016_MATRIZ.md).

## Correções

- Lookup de perfil e chave distingue `ErrNotFound` de indisponibilidade do store:
  continua 404 para perfil ausente e 401 para chave inválida; falhas internas
  respondem 500 genérico. O teste com erro injetado falhou antes da correção.
- Uma leitura antiga de “tocando agora” podia remover uma atualização recente
  do mesmo perfil após receber `ErrNotFound`. O teste reproduziu a falha com
  faixas diferentes e com o mesmo ID. O cache agora associa versão a cada `Set`;
  leitura antiga e confirmação de catálogo removem somente a versão observada.
  A expiração em `Get` é verificada e aplicada sob o mesmo lock. A confirmação
  captura as versões antes da mutação e não remove atualizações posteriores.
- Upload JSON de importação retornava 400 mesmo quando gravou registros e falhou
  ao arquivar. A regressão reproduziu 400; agora erro de arquivo inválido é 400,
  falha operacional é 500 e a mensagem continua alertando sobre resultado parcial.
- Aliases e relações de mídia sem relação com o perfil retornavam 500; a matriz
  HTTP reproduziu o caso. Agora retornam 404 para ausência, sem expor metadados
  de outro perfil; falhas de infraestrutura continuam 500.

## Matriz e evidências

`python3 scripts/validate_isolation_019.py --binary /private/tmp/koito-issue019`

Executado contra servidores HTTP reais em modo React e Go, com cópias novas da
fixture 016, SQLite v3, fontes externas desativadas e dados fictícios. Antes de
mutações, os oráculos independentes são Alpha=3, Beta=2 e Empty=0; após uma
escuta adicional autenticada como Alpha: 4/2/0. Os mesmos valores foram conferidos
por perfil no banco. `integrity_check` e `foreign_key_check` passaram.

- API: home/stats, rankings de artistas/álbuns/faixas com IDs e totais literais,
  histórico, mídia e relações/aliases, busca textual e por ID, atividade,
  interesse, summary e now-playing. Entidades exclusivas ou sem escutas foram
  negadas ou ausentes de buscas pessoais. Filtro de faixa Beta em Alpha retornou
  zero; cookie de admin e chave Beta não trocaram o perfil da URL Alpha.
- HTML Go: home com valores 3/2/0; páginas, histórico e busca pessoal sem mídia
  exclusiva alheia; query com ID conflitante rejeitada; busca global de destino
  disponível na gestão autenticada e indisponível ao visitante. Comparação
  numérica completa dos gráficos HTML permanece para a 021.
- Exportação estática em outra base: Alpha=1005 registros distintos, inclusive
  empate na fronteira entre páginas; Beta=2. Importação do export Alpha em Empty
  gravou 3 no destino e preservou os totais de Alpha/Beta. Export concorrente e
  consistência dinâmica continuam na 020.
- Mesma faixa/instante em dois perfis continua em ambos; repetição pessoal é
  ignorada e campo de perfil forjado no payload não altera destino. Chave
  revogada falhou e a de outro perfil continuou válida; dados persistiram no
  reinício e cache não persistiu.
- Dois servidores de relay locais receberam apenas o token e payload do perfil
  correspondente. Nenhuma escuta ou credencial real foi enviada.
- Duas abas abertas antes das ações preservaram tickets e perfil de destino;
  formulário aberto antes da renomeação retornou 409 sem reverter o nome e,
  após exclusão, outro retornou 404. Slug reservado foi recusado. Rotas legadas
  foram testadas com muitos, um e zero perfis (409/200/409).
- Remover e mesclar faixa pelo JSON limpou a origem de now-playing na próxima
  leitura, sem apagar a reprodução do outro perfil. Excluir perfil revogou sua
  chave e preservou o cache de outro. O teste de confirmação HTML cobre a
  invalidação condicional por versão, inclusive atualização concorrente e ABA.

Execuções adicionais:

- `go test ./...` e `go test -race ./internal/memkv ./internal/listening
  ./internal/admin ./internal/db/sqlite ./engine/middleware ./engine` passaram;
  alguns resultados utilizaram cache. `go vet` dos pacotes afetados passou.
- `TestProfileScopeAndCatalogEscape` verificou leitura pessoal de Alpha/Beta/Empty
  e o escape global interno; foi executado separadamente também com `-race`.
- `validate_multiuser.py` passou com bases nova e migrada v2→v3, ingestão,
  revogação, importação/exportação, reinício e integridade. A jornada
  `validate_admin_webui.py` passou nos cinco formatos, relay local e regressões
  administrativas anteriores. Nenhum banco em operação foi acessado.
- Advisor (Sol, esforço médio) aprovou as correções e revisou a matriz. A execução
  do validador HTTP é evidência do agente principal; o Advisor executou testes
  direcionados de SQLite e, com race, dos componentes de cache e middleware.

## Fronteiras e limites

Catálogo, aliases e capas são compartilhados globalmente por projeto; contagens,
escutas, chave de ingestão, export e relay são pessoais. Contexto sem perfil no
store ainda equivale à leitura global `WithoutProfile`, usada por tarefas internas;
o isolamento público depende de todos os adaptadores resolverem o perfil. Não foi
identificado caminho público que ignore essa resolução. Isso não é um controle de
acesso por perfil privado: os perfis continuam públicos.

A mutação de catálogo pela confirmação HTML invalida entradas observadas de forma
condicional. No JSON, a origem removida é limpa quando consultada e uma faixa de
destino existente pode continuar tocando; essa diferença de cache foi preservada
e testada, sem prometer entrega de now-playing em tempo real. Atualizações que
entram após a captura pré-commit são preservadas; se apontarem para item removido,
a próxima leitura elimina apenas a versão inválida.

O corpo ListenBrainz sem orçamento de bytes antes do parse, falhas/cancelamento
concorrentes e transporte lento permanecem na 020. O script não certifica toda a
matriz de atores/períodos e não valida numericamente todos os gráficos HTML; a
revisão transversal final pertence à 021. As limitações históricas de fuso e data
inteira ausente registradas na 018 permanecem.
