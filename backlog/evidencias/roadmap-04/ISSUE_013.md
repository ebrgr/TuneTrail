# Issue 013 — edição de mídia, imagens e escutas

**Atualização na issue 015:** integração Go, jornadas reais e comparação visual
registradas em [ISSUE_015.md](ISSUE_015.md). As notas abaixo descrevem a entrega
anterior à integração; o aceite final do usuário permanece pendente.

Em andamento em 05/11/2026. [Issue](<../../tasks/issue-013 - 013-—-Migrar-edição-de-mídia-imagens-e-ações-de-escutas-para-Go.md>) · [Plano](<../../docs/roadmaps/doc-007 - Roadmap-4-—-administração-importação-chaves-edição-deleção-e-mesclagem.md>).

O handler administrativo Go isolado prepara as rotas `/{slug}/manage/{kind}/{id}`
para aliases, MBID, compilação, busca e relação de artistas, sugestão e troca de
capa, adição de escuta e confirmação de exclusão das 25 escutas recentes.
Ações conferem sessão, CSRF e perfil por ID/slug, e o formulário avisa que o
catálogo é compartilhado. O dispatcher de produção permanece na interface atual.

A API JSON e os formulários compartilham os serviços de aliases, capa e escuta;
a operação de relação usa a mesma transação SQLite. A adição de artista não usa
outra conexão dentro da transação. Uma relação ausente não desmarca o artista
principal, e removê-la não executa a limpeza global que apagava faixas sem escutas.
Alias principal não é removido; ausência e conflito são identificados.

A substituição de imagem aceita upload ou URL HTTP(S), com até 10 MiB de imagem,
10 segundos e até três redirects validados; imagens até 10 mil pixels por dimensão
e 40 milhões de pixels. Destinos self-hosted continuam permitidos. A publicação
é coordenada com o coletor, arquivo novo é compensado em falha e a imagem anterior
fica para coleta global. Para uma faixa, o `UPDATE` confirma que o álbum não mudou
durante o download. A API JSON usa esse mesmo serviço.

A entrada de escuta manual interpreta o fuso IANA no servidor, rejeita o horário
inexistente e exige offset quando há ambiguidade de DST. A precisão suportada é
por minuto; offsets históricos com segundos são rejeitados. Exclusão no store
é por perfil/faixa/instante, com ausência identificada; a API JSON mantém DELETE
idempotente para compatibilidade. Erros e timeouts da operação de capa não revertem
uma gravação já confirmada.

## Testes

- SQLite v3 em memória: um único pool/tx, falha injetada e rollback, relação
  ausente e principal preservado, remoção de alias e preservação de faixa sem escutas.
- Perfis com a mesma faixa e timestamp: exclusão não afeta o outro perfil.
- Serviço de capa: imagem compartilhada, erro de banco, tipo inválido, redirect
  válido/inválido e faixa/álbum com IDs diferentes; alteração de álbum durante download.
- Handler Go isolado: sessão, CSRF, escopo, alias Unicode, perfil obsoleto,
  adição/duplicata de escuta; testes de lacuna e ambiguidade de horário de verão.
- `go test ./...` e testes de corrida dos pacotes admin/adminui/adminweb/SQLite
  passaram após os ajustes. As validações multiusuário e Go/React passaram em
  bases novas e migradas v3. Testes usam dados e imagens fictícios, sem contato
  com serviços externos.

## Ainda pendente

Concluir proteção contra cliques repetidos, integração dos caminhos de histórico
sem JavaScript, paginação/retorno e restante dos testes de aceitação. Valores
digitados e orientação de DST são preservados ao apresentar erro; arquivos devem
ser selecionados novamente pelo navegador.
A revisão visual e o arraste real em navegador permanecem para o final da 015.
O Advisor (Sol) revisou as correções implementadas e as aprovou, sem encerrar a issue.
