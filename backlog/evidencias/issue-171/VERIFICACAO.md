# ISSUE-171 — verificação do modo sombra

## Decisão aceita

Para itens ListenBrainz sem `listened_at`, o recibo usa a hora de chegada com precisão `estimated`. A escrita legada continua calculando o instante por item. Os horários podem divergir em lotes que cruzam a virada de segundo, preservando a chave de deduplicação usada em `listens`.

## Fluxo verificado

- A primeira escrita do lote serializa a consulta de chaves e candidatos entre conexões SQLite. O recibo registra decisão, motivo, evidência e regra. A resolução fica na versão zero e os IDs futuros ficam nulos.
- A API confirma o lote após o commit do recibo e a ingestão legada. Erro de disco na gravação do recibo ou falha no commit gera 503 antes de tocar o catálogo. Uma falha interna de sombra anterior ao commit é registrada em log e segue pelo caminho legado, conforme a exceção solicitada. `playing_now` pula o ledger e o relay continua no fluxo existente.
- Upload, importação manual e importação de inicialização passam pelos mesmos adaptadores. O SHA-256 dos bytes e o ordinal absoluto identificam cada checkpoint, inclusive quando o arquivo muda de nome. O cursor lê os snapshots já recebidos sem acesso à área de staging. Se o arquivo for reenviado, o parser reprocessa o caminho legado com as mesmas regras e o ledger reutiliza os recibos. A retomada da aplicação de eventos a partir desses checkpoints fica com a ISSUE-134.
- O contador `Pending` é uma leitura do resultado de sombra para os itens do upload. Pode sobrepor `Imported` e `Duplicates` e preserva as pendências de sombra durante reenvio equivalente.
- A chave opcional do cliente é escopada por perfil, instância, operação e ordinal. O adaptador ListenBrainz compatível só utiliza essa chave quando fornecida no cabeçalho `Idempotency-Key`. O armazenamento aceita IDs externos confiáveis com namespace explícito quando fornecidos por um adaptador. O payload compatível atual não declara tal ID.

## Testes de aceite

| AC | Evidência |
| --- | --- |
| 1 | `TestShadowReceiptACK` persiste dois recibos e uma decisão de Revisar para duas entradas sem chave no mesmo segundo. `listens` contém uma escuta. O mesmo teste cobre fail-open e exclusão de campos não autorizados do snapshot. `TestShadowFileResume` cobre digest, ordinais, cursor e contadores legados. |
| 2 | `TestShadowReceiptACK` injeta falha de commit por FK adiada, comprova 503 e confirma sucesso após reenvio. `TestShadowDiskFull` usa `max_page_count` para provocar SQLITE_FULL, observa rollback integral e reenvia o lote. O ACK de `playing_now` só atualiza o fluxo efêmero. |
| 3 | `TestShadowWALRetry` disputa a mesma chave com dois pools WAL e guarda a observação da variante conflitante. `TestShadowRepetitions` cobre mesmo instante com offset diferente, a hora repetida na volta do horário de verão e ordinais de arquivo independentes. `TestShadowClientKey` cobre reenvio de cliente, extensão conflitante de lote e isolamento por perfil. |
| 4 | Replay em cópia privada do banco real descrito abaixo. |
| 5 | `TestShadowWALRetry` aplica até a migração 004 e confirma uma linha com `duplicate_decision='same'` e `event_id IS NULL`. |

## Medição em cópia privada do histórico real

Os helpers de `scripts/validate_ingest_migration.py` calcularam em memória os hashes físicos do banco de origem e de seus arquivos WAL, SHM e journal, quando presentes. Depois copiaram esses componentes para um diretório temporário antes da primeira conexão SQLite. Somente a cópia foi aberta, validada e atualizada da versão 5 para a 6.

O replay projetou título, artista principal, lançamento, MBID, duração e instante de cada escuta preservada no histórico. Os itens foram ordenados por perfil, instante e faixa, e enviados em lotes de até 250 ao `ReceiveBatch`, como entregas sem chave confiável. A janela foi de ±2 s e a regra foi `shadow-v1`.

| Perfis com histórico | Escutas avaliadas | Diferente | Igual | Revisar | Taxa de Revisar |
| ---: | ---: | ---: | ---: | ---: | ---: |
| 1 | 80 | 80 | 0 | 0 | 0,0% |

As 80 linhas em `listens` da cópia mantiveram contagem e digest de conteúdo. `integrity_check` retornou `ok` e `foreign_key_check` não retornou linhas. Os hashes físicos da origem coincidiram antes e depois. O diretório temporário foi removido e só os agregados foram registrados.

A amostra mede colisões recuperáveis a partir do histórico salvo. Ela não cobre reenvios ou repetições no mesmo segundo já descartados pela chave legada, nem mede o tráfego futuro de fontes cruzadas. Os testes de aceitação exercitam essas colisões separadamente.

## Verificações locais

- `TUNETRAIL_WEB_DIR="$PWD/assets" make check` passou, incluindo Go vet/test, UI e 13 de 13 jornadas e2e.
- `CGO_ENABLED=1 go test -race -count=1` passou em `engine`, `engine/handlers` e `internal/db/sqlite`. Os pacotes `internal/importer`, `internal/ingest` e `internal/dedup` foram compilados com o detector e exercitados pelos testes integrados. O recorte `TestShadow` também passou separadamente.
- Graphify AST foi atualizado a partir desta worktree. `git diff --check` passou.
