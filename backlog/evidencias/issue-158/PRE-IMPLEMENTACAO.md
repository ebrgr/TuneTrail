# ISSUE-158 — pontos para revisão antes da migração

Registro histórico da inspeção inicial; as dúvidas abaixo foram resolvidas pelo parecer do Advisor e pelo aceite do usuário. Implementação e resultados em `VERIFICACAO.md`.

Inspeção em 18/11/2026. Nenhuma migração ou conexão com banco real executada.

## Contrato conferido

doc-041 §6 e decision-010: vínculo canônico 1:1, público por padrão, preservar admin/hash/dados/sessões e criar contas pendentes de ativação para perfis sem usuário. A ativação será emitida sob demanda pelo admin na ISSUE-162; não gerar senha temporária previsível, compartilhada ou publicada em logs.

## Coordenação de versão proposta, ainda não aplicada

ISSUE-080 está Done e publicou a v4 em 0.34. ISSUE-133 está Aberta, sem versão reservada. Proposta para o Advisor: 158 usa v5; registrar na 133 que sua migração usará a próxima versão disponível quando implementada. Não implementar pipeline, event_id ou identidade musical nesta tarefa.

## Dúvida de escopo submetida ao usuário

- `internal/db/sqlite/profile_store.go:165`: CreateListeningProfile cria somente perfil/chave; com FK obrigatória precisará criar também conta pendente, em transação.
- `internal/db/sqlite/profile_store.go:204`: DeleteListeningProfile apaga somente perfil. Manter esse comportamento deixaria conta sem perfil. Excluir a dupla também exigirá preservar o último admin e suas sessões.
- `engine/adminui_catalog_review_test.go:135`: teste atual exige apagar inclusive o perfil do admin e conservar sua conta/sessão. Esse comportamento é incompatível com o contrato 1:1.
- `engine/engine.go:148`: bootstrap consulta CountUsers e grava usuário/chave separadamente; exige operação transacional de bootstrap e propagação dos erros, sem recriar admin em base com contas.
- `internal/db/sqlite/profiles.go:24` e `profile_store.go:132`: compatibilidade de UserID e associação inicial de relay leem a ponte legada; devem passar a usar o vínculo canônico, preservando os vínculos já gravados.

Confirmar com Advisor se as adaptações mínimas de criação/exclusão na persistência e proteção do último admin entram na 158, mantendo autenticação/autorização/telas nas 159–163. Nenhum SQL novo deve ser escrito antes da coordenação da versão e da resolução desta dúvida.

## Verificação prevista após a revisão

Baseline executada: `go test ./internal/db/sqlite ./internal/admin ./engine`. SQLite e admin passaram; engine falhou no setup porque o servidor de saúde em `http://:63409/apis/web/v1/health` não ficou acessível (connection refused). Nenhum código foi alterado. `make check` ainda não executado; nenhum AC comprovado.

Testar cópias privadas v3/v4 e legado, backup consistente com WAL e restauração isolada antes da migração; falha real no meio da transação; reinício idempotente; comparação de hashes/IDs/dados existentes em memória, sem expor segredos; contagens por perfil; FK/UNIQUE/NOT NULL/default e cardinalidade; contas sem perfil, colisão de login/slug e base com contas sem admin. Rodar make check e testes de concorrência do bootstrap. Marcar ACs somente após comprovação. Manter Em andamento e sem push/merge/publish/Done.
