# ISSUE-143 — Excluir item só do perfil

## Resultado e regras exatas

O ícone Excluir e a página `/delete` abrem diretamente a confirmação com a pergunta “Você realmente deseja excluir o item \"NOME\"?”, o aviso de que não existe desfazer e a contagem de escutas calculada no servidor. Confirmar remove somente escutas do perfil da rota e navega para `/{slug}`, nos três tipos. Catálogo, aliases, imagens e escutas de outros perfis permanecem intactos. Cancelar e Esc fecham o diálogo e devolvem o foco; a página completa também funciona sem JavaScript.

Decisões do usuário recebidas pelo coordenador e registradas na tarefa:

- **Faixa:** nome principal da faixa e conjunto completo dos nomes principais de todos os artistas creditados à faixa.
- **Álbum:** nome principal do álbum e conjunto completo dos nomes principais de seus artistas. É usada a mesma relação canônica da página: créditos independentes/legados unidos à participação derivada das faixas (`albumArtistRelations`).
- **Artista:** nome principal do artista, mas somente faixas com **um único crédito**. Colaborações não são removidas, mesmo que incluam o artista; o diálogo explica essa limitação. Outros IDs de artistas com o mesmo nome principal também correspondem quando são o crédito único.
- Igualdade textual SQLite `COLLATE BINARY`: sensível a caixa, espaços, acentos e representação Unicode, sem normalização. Aliases secundários não correspondem. A ordem e a marca de artista primário/secundário na relação não afetam o conjunto de nomes; nomes repetidos formam um conjunto, não uma concatenação. A+B corresponde a B+A, mas não a A, B ou A+C. Conjuntos vazios correspondem apenas a conjuntos vazios. Para artista, “crédito único” conta as relações reais, não os nomes distintos.

O mesmo seletor SQL gera a contagem da prévia, a revalidação e os registros do DELETE. Uma confirmação desatualizada não remove a contagem antiga nem inclui silenciosamente escutas novas: responde conflito e exige abrir uma confirmação nova.

## Desenho e segurança

`internal/db/sqlite/profile_delete.go` concentra a seleção por nomes e a transação. A prévia lê um snapshot consistente; a confirmação abre **uma transação** que recalcula o snapshot, compara o fingerprint, executa um único `DELETE FROM listens` com `profile_id` explícito, verifica `RowsAffected` e faz commit. O fingerprint cobre o perfil, tipo/ID da rota, nome, conjunto de artistas e as escutas selecionadas (identidade, instante e cliente), em memória constante. Mudanças de créditos também invalidam a confirmação. Nenhum DELETE de artista, álbum, faixa ou alias é usado neste fluxo.

A sessão administrativa é resolvida novamente em cada requisição. Origem/Fetch Metadata e CSRF continuam passando pela proteção do handler principal. O plano de dez minutos é ligado a sessão, ID/slug do perfil, tipo e ID da rota e consumido uma vez. A exclusão pessoal também exige o ticket de submissão de uso único em produção. IDs/slugs enviados no formulário precisam coincidir com o perfil resolvido da rota; nenhum campo de formulário escolhe o escopo da consulta. A resposta inclui `X-TuneTrail-Deleted-Listens`; diálogos recebem 204 e `X-TuneTrail-Navigate`, páginas completas recebem 303.

Erros de seleção, validação, trigger, cancelamento, locks ou commit não deixam exclusão parcial. Corridas de snapshot/escrita SQLite são tratadas como conflito; não há retry da mutação. O cliente já existente apresenta erro traduzido e acesso à página completa, sem repetir um POST de resultado incerto.

A rota antiga `/review?action=delete` agora emite a confirmação **pessoal**, inclusive com ticket. A revisão global no HTML aceita somente merge; foram removidos a página de revisão global de exclusão e seu bloco de impacto morto. As rotas JSON de exclusão global e o código de catálogo foram mantidos porque ainda fazem parte da API administrativa existente; o ícone e as páginas HTML não oferecem essa operação. A mesclagem mantém seu plano e comportamento anteriores.

## Causa raiz do fluxo anterior

Reprodução em Chrome com binário, assets e idiomas da `main` v0.22 anteriores à implementação, usando somente fixture descartável. O formulário GET “Revisar exclusão” tinha `<input name="action" value="delete">`. Esse controle encobria `HTMLFormElement.action`; `new URL(form.action)` recebia o elemento em vez da URL e lançava **“Failed to construct 'URL': Invalid URL”**. Não era feita sequer a requisição de revisão: zero chamadas `/review`. O diálogo ficava parado, sem erro visível. O cliente agora lê `form.getAttribute('action')` com base na URL atual, tanto no envio quanto nas buscas.

Foi investigada também a confirmação final: o checkbox obrigatório não marcado bloqueava nativamente o envio. Abrindo diretamente a revisão e marcando o checkbox, o POST anterior retornou 204, navegou para `/alpha` e efetivamente apagou a faixa **globalmente**, removendo também uma escuta de beta. Portanto, não foi reproduzida uma falha do DELETE no banco depois de uma confirmação final válida; a falha reproduzida é anterior ao envio, na construção da URL. O novo fluxo dispensa esse passo intermediário e seu checkbox, com a confirmação explícita no botão “Excluir escutas”. Evidência: `diagnostico-anterior.json`; nenhuma instância operacional foi alterada.

## Verificação

- `make check`: aprovado, incluindo vet, testes Go, JavaScript/estilo, referência HTML/CSS e **12/12 jornadas E2E**. Saída em `make-check.txt`.
- `engine/adminui_profile_delete_143_test.go`: roteador principal com tickets obrigatórios; dois perfis com nomes iguais; outros IDs com os mesmos nomes; artistas diferentes e aliases secundários; A+B/B+A/A+C; caixa e espaços; créditos derivados do álbum; artista com créditos mistos; contagem exibida igual à removida; outros perfis e todo o catálogo preservados; perfil sem escutas; IDs/slugs forjados; ID inexistente; visitante; sessão expirada; origem estrangeira; CSRF; ticket ausente e replay isolado do plano; replay do plano; alteração posterior de escutas/créditos; mudança em outro perfil sem ampliar a seleção; trigger que falha no meio do DELETE com rollback completo.
- `scripts/validate_profile_delete.cjs`: Chrome instalado, em fixture nova e porta efêmera. Clique real nos três tipos, contagem/resposta/estado concordantes e navegação para o perfil; preservação das colaborações; foco preso, Esc, foco devolvido e Cancelar; rede sem retry; sessão expirada no banco entre abertura e confirmação; resposta perdida após POST processado sem repetição; fallback sem JavaScript e zero escutas. Resultado em `browser-checks.json`.
- Doze capturas dos diálogos: `track|album|artist` × `yuu|pearl` × `desktop|mobile`, além de `falha-rede.png` e `sessao-expirada.png`. Conferidos visualmente desktop Yuu e celular Pearl; sem overflow nas medições dos doze estados. O aceite visual pertence ao usuário.
- Referência regenerada em diretório novo por `make ui.reference OUT=/tmp/tunetrail-reference-143-ready` e copiada para `backlog/evidencias/issue-067/reference`. Mudanças públicas limitadas aos hashes dos assets alterados. CSS na camada `components`, arquivo específico do diálogo, abaixo de 400 linhas, usando os tokens dos temas.
- Deslop: reutilizados o diálogo, o shell administrativo, os planos limitados e os tickets existentes. Sem novo mecanismo de sessão, retry, exclusão global ou tabela persistente. `git diff --check` sem problemas.
- Graphify atualizado pelo script AST oficial, sem extração semântica/custo de API. O script só reconhece `.git` como diretório; na worktree ele é arquivo. Foi executado o mesmo script por um wrapper de shell que reconhece esse arquivo no teste inicial, sem editar o script nem metadados Git. Saída final: 3413 nós; `tokens.css` foi ignorado como potencialmente sensível pelo extrator, como informado por ele.

## Revisão, riscos e aceite

Por decisão explícita do usuário transmitida pelo coordenador durante a execução, **não foi chamado o Reviewer via Claude CLI**. A revisão do diff e o Advisor ficam com o coordenador após esta entrega; não há parecer externo produzido por este worker.

Riscos para essa revisão: a exclusão é irreversível e baseada em nomes atuais, incluindo outros IDs com os mesmos nomes; renomeações/escutas concorrentes exigem nova confirmação; um histórico muito grande exige leitura linear das escutas selecionadas e manutenção do lock durante a transação, embora o fingerprint seja incremental e não armazene a lista em memória. Não foram criadas tombstones: novos scrobbles/importações podem inserir novas escutas depois de excluir. Colaborações são deliberadamente preservadas ao excluir artista. Uma resposta perdida pode deixar o usuário sem saber se houve commit, por isso não se repete a mutação; abrir a tarefa novamente consulta o estado atual.

Não houve uso da porta 4110, leitura de `.env` ou bancos do usuário, nem alteração do Koito em 4111. Todos os servidores e bancos de teste foram descartáveis. Fora do escopo: × do histórico (ISSUE-141) e arrastar capa (ISSUE-142). Sem push, merge, publicação ou Done.

Critérios #2–#6 verificados. O #1 permanece desmarcado para o aceite visual do usuário: conferir a janela pequena, o nome/pergunta, o aviso irreversível, a contagem e a explicação das colaborações, em especial Pearl/celular. Falta também a revisão do diff pelo coordenador.

Entrega: `make issue.review ISSUE=143` executado com sucesso e gate repetido (12/12 E2E); tarefa em **Aguardando aceite**. Critérios #2–#6 marcados; #1 permanece para o usuário. Nenhuma revisão externa foi executada, conforme a instrução do coordenador.

## Rebase sobre a v0.23 (16/11/2026)

A ISSUE-141 foi publicada (v0.23) depois que esta branch foi entregue. O coordenador reaplicou a branch sobre a `main` atual (`git rebase main`): só conflitaram o `CHANGELOG.md` e a referência HTML da ISSUE-067 (arquivo gerado, regenerado com `make ui.reference`); os arquivos de código se mesclaram sem conflito e foi conferido que a janela da escuta e o × da 141 e a exclusão por perfil da 143 permanecem. `make check` verde, 12/12 E2E. As validações em Chrome desta nota não foram repetidas após o rebase, por decisão do usuário; valem as da entrega original.
