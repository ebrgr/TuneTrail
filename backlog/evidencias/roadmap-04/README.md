# Roadmap 4 — evidências do estudo

Estudo em 05/11/2026. [Plano](<../../docs/roadmaps/doc-007 - Roadmap-4-—-administração-importação-chaves-edição-deleção-e-mesclagem.md>).
**Atualização: issues 010–015 integradas e testadas; 55 imagens comparativas registradas.**
[Entrega e validações da 010](ISSUE_010.md) · [regressões corrigidas](issue010-regressions.json) · [entrega e validações da 011](ISSUE_011.md) · [entrega e validações da 012](ISSUE_012.md) · [andamento da 013](ISSUE_013.md) · [entrega da 014](ISSUE_014.md).
Layout e comparação visual foram realizados na 015. [Entrega final e limites](ISSUE_015.md) · [manifesto visual](screenshots.json).

O restante descreve o estudo inicial. A análise leu o backend, cliente e documentação atuais; não implementou páginas,
serviços ou correções de runtime. O serviço em uso permaneceu parado.

## Reprodução isolada

[analysis-checks.json](analysis-checks.json) registra chamadas diretas aos métodos
Go reais com bancos independentes `sqlite.NewInMemory()`, esquema v3. Cada banco
recebe dois perfis, dois artistas/álbuns/faixas e quatro escutas fictícias.

- `MergeArtists(ctx, 1, 1, false)` retorna nil; escutas passam de quatro para duas,
  afetando ambos os perfis que ouviam a origem. O artista é removido.
- `MergeAlbums(ctx, 1, 2, false)` com a conexão única de `NewInMemory` não conclui
  antes do contexto de 250 ms; retorna `context deadline exceeded` e conserva
  quatro escutas. A transação retém a conexão enquanto o helper tenta outra.

Isso comprova problemas dos métodos de persistência nas condições acima. Não
comprova uma exploração remota, falha HTTP ou bloqueio do pool da configuração
usual de produção, que tem outros parâmetros. Não é benchmark de duração do merge.

A fonte utilizada está preservada em [reproduction.go.txt](reproduction.go.txt).
Para repetir, copiar para `.local/roadmap04-study/main.go` e executar a partir da
raiz: `GOCACHE=/private/tmp/koito-ui-go-cache go run ./.local/roadmap04-study`.
O programa abre apenas bancos em memória; não usa `.env` nem o banco do usuário.
As migrações aplicadas são as 001–003 vigentes. Não é uma correção dos problemas.

## Revisão do Advisor

O Advisor leu `agents.md` e sua persona e revisou a proposta no modelo Sol, esforço
alto. Parecer final: **aprovado**, após incorporar as ressalvas e conferir o
plano 010–015. Foram incorporadas: separar correções funcionais da apresentação; CSRF com proteção
pré-login e vínculo de sessão; consultas e revalidação dentro do tx; importação
síncrona/parcial; limites de rede/arquivo e isolamento de segredos. A revisão
final também delimitou confirmação/CSRF das novas actions HTML para preservar
os payloads JSON atuais, validação de CSRF antes de spool multipart e ausência
de relatório recuperável após desconexão. Seus achados
foram de leitura; as duas reproduções acima foram executadas pelo agente principal.

## Evidência ainda futura

Não houve execução da suíte completa, navegador, chamadas externas, testes HTTP,
mutação no banco operacional ou captura visual nesta etapa de estudo. Também não
houve publicação de issues, commit, push ou mudança na configuração de execução.

As issues 010–015 especificam os testes futuros. Pares devem incluir login/conta,
perfis/avatar/chaves/relay, import/export e gestão de catálogo, em cópias equivalentes
SQLite v3. Criar `screenshots/` e manifesto somente durante a implementação; imagens
da etapa 3 não comprovam que a administração Go já exista.
