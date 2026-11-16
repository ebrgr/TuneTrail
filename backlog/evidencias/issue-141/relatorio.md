# ISSUE-141 — Escuta individual no histórico com página e diálogo

Implementação na branch dedicada `feat/issue-141-escuta-individual-no-historico-com`, sobre a v0.22. O comentário #2 da tarefa substitui a decisão antiga sobre o × rápido: agora ele abre a confirmação nativa, e não envia mais DELETE direto à API.

## Comportamento entregue

Na home do perfil, `/listens` e páginas de faixa, álbum e artista, o × abre um `<dialog>` com o mesmo fragmento da página individual da escuta. A janela pergunta se deseja excluir e avisa expressamente que não existe desfazer. Mostra faixa, artistas e álbum com links, data/hora no fuso efetivo da página e cliente original (`client`); quando esse campo não foi registrado, mostra “Não registrado”. A checkbox e o botão confirmam a remoção apenas daquela escuta.

O link Gerenciar é renderizado apenas para administradores. Com JavaScript ele fica oculto e sua URL alimenta o ×; sem JavaScript permanece visível e abre a página de confirmação existente. Visitantes não recebem Gerenciar, × nem os assets administrativos do diálogo. O carregador da ISSUE-140 é reutilizado nas páginas que contêm histórico, sem carregar `admin.js`.

O POST conserva sessão, origem/Fetch Metadata, CSRF, ticket de uso único e perfil resolvido pela rota. A exclusão usa perfil, faixa e instante; o teste mantém a mesma faixa e instante em outro perfil. Uma resposta 204 reconhecida encerra aquele formulário; o cliente busca o estado atualizado pelo renderer público e substitui o conteúdo principal, mantendo a URL e o documento, restaurando a rolagem e levando o foco ao próximo ×, ao anterior ou ao título quando o histórico fica vazio. Assim também se atualizam estatísticas, contagens, paginação, rankings e gráficos. Eventos de período e falha de capa são delegados, funcionando também após a troca do conteúdo.

Ao remover a última escuta de uma mídia, somente a atualização autenticada pode renderizar metadados globais com contagens pessoais zeradas e histórico vazio; a navegação pública normal continua respondendo 404 para um item não ouvido no perfil. As leituras de histórico, rankings, gráficos e atividade continuam no contexto pessoal. Sem JavaScript, o retorno é a página de origem quando ainda existe; se ela perder sua última escuta, vale o fallback protegido da ISSUE-139. `history_return` agora aponta para a própria página de mídia e retém o fuso efetivo.

Falha de rede, sessão expirada e respostas inesperadas não inserem documentos completos no diálogo nem repetem mutações. Se a exclusão for reconhecida e a atualização do conteúdo falhar, a linha selecionada é removida e a janela informa que a mudança foi salva e oferece recarregar o histórico; as demais contagens exigem essa recarga. Fechar a janela cancela leituras pendentes. Suspensão/retorno pelo histórico limpa a autoridade, remove controles e revalida a sessão antes de recriá-los. Rolagem é preservada até o limite físico do documento quando seu tamanho diminui.

## Verificação

- `make check`: aprovado, incluindo vet, testes Go, sintaxe de JavaScript, estilo, referência HTML/CSS/JS e **12/12 jornadas E2E**. Os testes usam fixtures descartáveis e portas efêmeras.
- `engine/adminui_listen_141_test.go`: roteador principal com tickets obrigatórios; conteúdo idêntico na página/fragmento, links, `client` escapado e horário em Asia/Tokyo; ausência de confirmação sem exclusão e com ticket renovado; exclusão e replay rejeitado; outro perfil com a mesma faixa/instante preservado; atualização autenticada vazia nos três tipos e 404 para visitantes; CSRF, ticket ausente, sessão, origem, perfil/slug forjados e retornos cruzados; retornos sem JS às três páginas de mídia; HTML e assets por papel nas cinco superfícies.
- `scripts/validate_listen_dialogs.cjs`: **Chrome 157.0.8103.51 real**, fixture descartável nova. Cinco superfícies × desktop/celular × Yuu/Pearl; detalhes, aviso, links, ausência de Gerenciar separado, Tab, Esc, foco devolvido e ausência de overflow. Também cobre resposta tardia após fechar, falha de rede/recuperação, confirmação obrigatória, atualização de linha/contagem sem novo documento, rolagem/foco, outro perfil preservado, estado vazio da última escuta da faixa, período lembrado após atualização, fallback de nova capa com falha, alternativa sem JS e sessão realmente expirada no banco. Resultado: [browser-checks.json](browser-checks.json).
- `scripts/validate_web_cache.py --browser chrome`: prova adicional com navegação real e bfcache; dois ciclos com um controle por linha, documento/URL/rolagem preservados, logout e expiração no servidor, modo offline sem controles, idioma alterado e exclusão confirmada após voltar com autorização nova. Resultado: [cache-chrome.json](cache-chrome.json). O script foi adaptado para confirmar a nova janela antes da exclusão.
- Referência da ISSUE-067 regenerada com `make ui.reference OUT=<diretório novo>` e copiada para `backlog/evidencias/issue-067/reference`. Não houve atualização manual dos arquivos gerados. `git diff --check` sem problemas; partes de CSS no limite de 400 linhas, nas camadas e temas existentes.

## Capturas e aceite visual

As 20 capturas `dialog-*.png` cobrem as cinco superfícies em dois tamanhos e dois temas. Exemplos da faixa:

| Tema | Desktop | Celular |
| --- | --- | --- |
| Yuu | [diálogo](dialog-track-1-yuu-desktop.png) · [página](pagina-yuu-desktop.png) | [diálogo](dialog-track-1-yuu-mobile.png) · [página](pagina-yuu-mobile.png) |
| Pearl | [diálogo](dialog-track-1-pearl-desktop.png) · [página](pagina-pearl-desktop.png) | [diálogo](dialog-track-1-pearl-mobile.png) · [página](pagina-pearl-mobile.png) |

Há capturas de [rede indisponível](erro-rede.png), [sessão expirada](sessao-expirada.png) e [última escuta removida](historico-vazio.png). As quatro capturas `pagina-*.png` mostram a página completa que também funciona sem JavaScript. As capturas foram inspecionadas para layout e overflow; a aprovação visual pertence ao usuário. Conferir principalmente a hierarquia da confirmação, detalhes em Pearl/celular, o aviso sem desfazer e o foco/retorno ao excluir.

## Revisão, grafo e limites

Por decisão explícita do usuário, repassada pelo coordenador, a revisão do diff será feita pelo **coordenador após esta entrega**; não há parecer externo registrado nem alegação de aprovação do Reviewer. Não foram usados subagentes ou helpers para a implementação.

O grafo foi consultado antes da exploração. O script oficial `analisar-tunetrail.sh` usa o checkout principal por padrão e exige `.git` como diretório, enquanto nesta worktree `.git` é arquivo. Por orientação do coordenador, a ferramenta externa não foi alterada e não foi feito rebuild sobre esta worktree; o coordenador atualizará o grafo na main depois da publicação. A execução inicial padrão não incluiu as alterações desta branch.

Não foi necessário acessar a instância Koito em 4111 nesta tarefa: os diálogos publicados e seus relatórios forneceram a referência de estilo. Nenhum servidor foi iniciado em 4110, nenhum banco operacional foi usado, e não foram gravadas credenciais ou sessões operacionais em evidências. As ISSUE-142 e ISSUE-143 não foram implementadas ou editadas. Não houve push, merge, publicação ou status Done.

Critérios funcionais #1–#6 verificados. O item #3 é interpretado conforme a decisão substitutiva do comentário #2: o × funciona com a janela e a confirmação, sem a exclusão rápida antiga. O item #4 comprova os testes e a produção das capturas; não representa aceite visual do usuário. `make issue.review ISSUE=141` repetiu o gate completo com **12/12 E2E** e colocou a tarefa em **Aguardando aceite**; faltam a revisão do coordenador e o aceite explícito do usuário para publicar.
