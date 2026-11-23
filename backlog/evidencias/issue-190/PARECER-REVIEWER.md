# ISSUE-190 — Parecer do Reviewer (23/11/2026)

Revisão dos commits a6f375c e 96e0ae0 feita por quem despachou, sem delegar. Lido o diff da ingestão, das consultas, da migração, das importações, da exportação e do relay, a medição e as correções pedidas.

**Veredito: APROVADO COM RESSALVA.** Um defeito foi encontrado na primeira revisão e corrigido em 96e0ae0, com a prova repetida. Resta uma decisão do usuário.

## Defeito encontrado e corrigido

- [BUG, corrigido] `LbzAdditionalInfo` lia `duration_played` e `duration_played_ms` como `*float64`. Um cliente que mandasse `"duration_played":"120"` fazia o `json.Unmarshal` falhar e a escuta inteira seria recusada, quando antes o campo era ignorado. Em 96e0ae0 a leitura aceita número e texto numérico e trata qualquer outro tipo como medição desconhecida. Prova repetida com um teste descartável (removido): `"120"`, `"90500"`, `"abc"`, objeto, lista, `true`, `null`, texto vazio, `120.9` e `1e400` leem sem erro.

## Verificado

- [OK] Conversão: segundos truncados, milissegundos divididos por 1000 e truncados, negativo, NaN e valor acima de `MaxInt32` viram desconhecido, e o zero explícito é preservado (`PlayedSeconds`).
- [OK] Gravação: o limite pela duração da faixa é feito na própria consulta (`min(?, coalesce(duração>0, 2147483647))`), em todos os caminhos de escrita (ingestão, revisão, restauração de arquivo). `min` com `NULL` continua `NULL`, então o desconhecido não vira número.
- [OK] Consultas de tempo ouvido de faixa, álbum, artista, período e Rewind usam `COALESCE(duração tocada, duração da faixa)`, e o caso de teste do PR 379 soma 410 s (120 + 90 + 200).
- [OK] Migração 023 aditiva, com `CHECK` de inteiro não negativo, `Down` recusado com `restore_verified_backup`, testes de migração e de loja. Os testes antigos que rodam em schemas históricos foram adaptados no mesmo modelo da ISSUE-202.
- [OK] Exportação e reimportação: o campo é opcional, o arquivo antigo sem o campo continua legível, o campo ausente continua ausente e um zero antigo vira desconhecido ao restaurar, e a reimportação valida a faixa antes de gravar. O relay envia `duration_played` inteiro em `additional_info`. O Spotify passa a guardar `ms_played` como medição e mantém a duração antiga como estimativa de catálogo.
- [OK] Medição honesta e reproduzível: o `ALTER TABLE` levou 85 ms em 300.080 escutas sem crescer o número de páginas; os planos de consulta e os tempos estão registrados, inclusive os recortes por item 0,3 a 1,8 ms mais lentos com a coluna toda nula.

## Riscos e decisão

- [DECIDIDO] Zero é ausente (decisão do usuário em 23/11/2026). Um `duration_played: 0`, uma medição de menos de um segundo, um valor negativo ou não numérico ficam como `NULL` e a escuta conta a duração inteira da faixa; se os segundos vêm zerados e há `duration_played_ms` válido, vale o milissegundo. A regra vale na leitura (`PlayedSeconds`) e na própria gravação em SQL (`NULLIF(?,0)`), então importações de arquivos antigos, o relay e a restauração de arquivo seguem a mesma regra. O relay não envia um zero. O `CHECK` da coluna continua aceitando zero, de propósito, para que uma escrita inesperada de zero nunca faça perder uma escuta; o total conta zero nesse caso, mas nenhum caminho do código grava zero.
- [RISCO] Os planos de consulta por faixa, álbum e artista passam a usar `idx_listens_track_id` em vez do índice composto, porque a consulta agora lê a coluna nova. É mais lento por 0,3 a 1,8 ms na amostra de 300 mil escutas; o ganho na consulta ampla de período (204 ms para 20 ms) vem da mudança de plano, não da coluna.
- [RISCO] A migração 023 roda uma vez no banco existente na primeira execução, com backup antes; não há volta pelas migrações. A ISSUE-195 deve incluir a 023, além da 022.

## Fora do alcance

- O aceite é do usuário.
