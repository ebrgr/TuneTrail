# Issue 022 — resultado

Implementada em 05/11/2026. [Matriz e oráculos](ISSUE_022_MATRIZ.md),
[protocolo](ISSUE_022_PROTOCOLO.md) e [manifesto modelo](manifesto-modelo.json).

A fixture versionada 1 foi criada a partir da fixture 016, com cópias SQLite v3
para Go/React e ações independentes, 0/1/N perfis, variante paginada, imagem
compartilhada, imagem ausente, Unicode e quatro instantes de DST com valores
literais em São Paulo. O usuário e chaves são fictícios; a senha é pública de
propósito. Administrador pré-criado evita que o bootstrap crie perfil extra.

## Validação executada

- `python3 -W error::ResourceWarning scripts/test_roadmap06_fixture.py`: 2 testes
  passaram; diretório existente foi preservado e mutação de uma cópia não alterou
  as outras. A geração também verificou schema v3, FKs, integridade, totais,
  dados DST e empate na fronteira de exportação. SQLite 3.54.1 nesta execução.
- `python3 -m py_compile` nos três scripts e `python3 -m json.tool` no modelo de
  manifesto: passaram.
- `go build -o /private/tmp/koito-issue022 ./cmd/api`: passou com cache local e
  downloads Go desabilitados. O iniciador `--smoke` levantou Go e React em loopback,
  confirmou modo e contagens Alpha/Beta/Empty 7/2/0 e encerrou os processos.
- Um ensaio HTTP com cópias descartáveis confirmou login fictício e contagens
  7/2/0 em ambos os modos, zero perfis, um perfil e variante grande 1009/2/0.
  O ensaio de login precedeu apenas a adição do arquivo de imagem; esta não muda
  a autenticação ou dados de perfil.

O relatório de fixture é produzido em tempo de execução; o hash identifica a
cópia gerada, mas não se promete hash idêntico entre gerações, pois metadados de
migração podem depender do relógio. A data de referência documentada **não altera
o relógio do processo**. Não se fez comparação visual, cliques em React hidratado,
validação v2→v3 para web/admin ou teste de cada intervalo: permanecem nas issues
023–025. Não foi alterado o runtime nem o banco operacional.
