# ISSUE-218 — implementação do modelo D

## Feito e decisões

- A migração 026 cria o cadastro de fontes e metadados descritivos por perfil e entidade, incluindo os tipos necessários à ISSUE-217. MBIDs e sua tabela por escuta permanecem inalterados; a migração precede a consolidação da ISSUE-195.
- Os valores de faixa pertencem à ocorrência estável: mesclar ou desfazer a mesclagem das gravações não confunde as observações. O catálogo não tem grupo de lançamento; `first_release_date` pertence à edição e representa a afirmação da fonte sobre o primeiro lançamento, sem inventar um grupo de catálogo. `first_recording_date` pertence à ocorrência.
- A fonte e a impressão digital ficam no cadastro separado. `observed_at` é o instante da primeira observação do valor pela fonte; a repetição não o substitui.
- Navidrome só consulta detalhes após correspondência inequívoca e não truncada; `getSong` e `getAlbum` são verificados de novo. A documentação [OpenSubsonic AlbumID3](https://opensubsonic.github.io/docs/responses/albumid3/) confirma `originalReleaseDate`; datas parciais mantêm sua precisão. ISRC chega ao MusicBrainz na cadeia de identidade pelo orçamento compartilhado, e resultados múltiplos são ambíguos.
- Registros de execução `found` persistem enquanto o trabalho pode ser recuperado e são apagados quando ele termina, na mesma transação e condicionados ao lease. Falhas e ambiguidades permanecem.
- O ensaio sintético final, incluindo fontes e inserção/remoção de execução sem VACUUM posterior, mede 671,33 / 220,36 / 65,54 B por escuta para 1 / 3 / 10 escutas por faixa; hipóteses e script em MEDICAO.md.
- Cobertos escopo por perfil, escuta, faixa, álbum, artista, perfil, expurgo, descarte protegido, mesclagem e reversão, respostas ambíguas e truncadas, repetição, falhas, retomada e perda do lease. Teste de migração da v25 preserva dados e verifica chaves e gatilhos.
- Testes de mutação: a associação segura e a limpeza de sucessos foram quebradas temporariamente; os testes correspondentes falharam e o código foi restaurado.

## Limitações e pendências

- A exportação atual não inclui os novos metadados descritivos nem restaura esses valores; a ISSUE-234 acompanha a solução. A instância da fonte não é exportada.
- A apresentação dos metadados é da ISSUE-212 e o provedor MusicBrainz de metadados é da ISSUE-217.
- Não foram usados servidores reais, dados pessoais ou credenciais. Não foi feita medição em banco de produção.
- A tarefa continua Em andamento; aceite, publicação, push, merge, tag e Done dependem do fluxo posterior autorizado pelo usuário.

## Verificação

- `make check`: aprovado na execução final, incluindo a suíte Go, lint, testes de scripts e 13 jornadas de ponta a ponta.
- `CGO_ENABLED=1 go test -race ./internal/enrichment ./internal/db/sqlite -run 'Test(Subsonic|MusicBrainz|Navidrome|Descriptive|Bootstrap.*Enrichment|BootstrapDescriptive|Enrichment|IngestDiscardProtectsListens)' -count=1`: aprovado; os últimos testes de integração foram repetidos separadamente com `-race`.
- O script de análise AST do graphify foi executado na worktree após as alterações de código.

## Parecer do Reviewer — correções

1. **Identidade e detalhes:** falha ou divergência em `getSong`/`getAlbum` preserva as declarações já seguras; divergência do álbum conserva os valores seguros da faixa. Respostas parciais não entram no cache. `metadata_retries` mantém a necessidade de nova tentativa por perfil, ocorrência e fonte mesmo após reinício; a última escuta remove o marcador, e uma resposta completa o limpa.
2. **ISRC inexistente ou indisponível:** 404 é ausência somente para o endpoint de ISRC; ausência e erro seguem para busca por nome. Se o ISRC falhou e a busca não encontrou nada, o resultado é falha e não é guardado como ausência.
3. **Vários ISRCs:** até quatro códigos ordenados compartilham o orçamento do MusicBrainz; gravações distintas são reunidas, e mais de uma é ambígua. Uma consulta falha impede declarar com base apenas nas demais; acima do limite segue para a busca por nome. Resultados conclusivos usam cache.
4. **Artistas:** `inc=artist-credits` recupera créditos na consulta por ISRC, com checagem de nome, unicidade e conflitos de IDs conhecidos. A [API MusicBrainz](https://musicbrainz.org/doc/MusicBrainz_API) documenta os includes para ISRC e gravação.
5. **Escala:** os quatro gatilhos separam vínculos e usam os índices por entidade e perfil/faixa. Consultas curtas próximas da faixa e no início do perfil evitam scans repetidos nos lotes medidos; o fallback pode percorrer as faixas do próprio artista ou lançamento quando as consultas curtas não encontram suporte. `MEDICAO.md` descreve os cenários de até 100 mil escutas e `PLANOS-GATILHOS.txt` contém os planos.
6. **Consulta condicionada:** o armazenamento identifica metadados de faixa da mesma ocorrência, perfil, fonte e instância e informa o provedor sem alterar a chave de cache. Com esses dados e sem IDs pendentes, não há rede; com ID pendente, ausência de metadados de faixa ou marcador de falha parcial, a consulta continua. Não há atualização periódica de valores já gravados nesta issue. Se o servidor não retorna metadados de faixa, escutas futuras voltam a consultá-lo.

Os testes de reprodução falharam antes das correções. As seis mutações compilaram e foram detectadas; há também mutação específica para a declaração de créditos. O registro resumido está em `MUTACOES.txt`.

### Resposta aos riscos

- **Lease vencido:** a tentativa por provedor tem timeout de 20 s, e cada etapa concluída renova o lease de 60 s. Se o lease vencer antes da gravação, o executor antigo não encerra nem apaga a execução alheia; o worker recupera após o vencimento, preservando os passos finais, com limite de duas tentativas e falha terminal ao esgotá-las. Os testes cobrem expiração, perda do lease e recuperação esgotada. Com o processo parado ou banco indisponível a recuperação aguarda o serviço voltar; não há garantia de conclusão enquanto isso não ocorrer.
- **Tipo concreto `*Subsonic`:** continua existindo na otimização que evita repetir metadados na mesma passagem. É acoplamento de extensibilidade; os testes atuais de recuperação e de uma consulta por passagem não demonstraram erro funcional que justificasse alterar o contrato de provedores nesta correção. Um segundo provedor com a mesma propriedade exigirá generalizar a capacidade.
- **Exportação:** o formato continua sem os metadados descritivos e a ISSUE-234 acompanha exportação/restauração. A instância da fonte não é exportada.

A entrada existente de `[Unreleased]` já descreve a funcionalidade ainda não publicada; as correções do parecer não acrescentam uma segunda entrada para regressões de código que ainda não foi lançado.

Verificação das correções: `make check` aprovado, com 13 jornadas de ponta a ponta; `CGO_ENABLED=1 go test -race ./internal/enrichment ./internal/db/sqlite -run 'Test(Reviewer|Subsonic|MusicBrainz|Navidrome|Descriptive|Bootstrap.*Enrichment|BootstrapDescriptive|Enrichment|IngestDiscardProtectsListens)' -count=1` aprovado. O teste de instância foi ampliado e repetido separadamente com `-race`. O grafo AST foi atualizado. Nenhum serviço real com credencial foi consultado.

## Segunda rodada de correções

- Divergências em `getSong` ou `getAlbum` são definitivas: preservam as declarações seguras, podem ser armazenadas no cache e não criam marcador de nova tentativa; uma divergência definitiva também remove um marcador anterior. Falhas de chamada continuam repetíveis.
- A chave do cache Subsonic inclui os campos de gravação e artistas pendentes após considerar `Prior`, evitando reaproveitar declarações incompletas.
- Antes das correções, os testes novos reproduziram ambas as falhas; após as correções passaram os testes direcionados:
  - `go test ./internal/enrichment -run '^(TestSubsonicPermanentDivergenceCachesSafeClaims|TestSubsonicCacheSeparatesPendingIdentifiers|TestReviewerSubsonicPartialIdentityRetries|TestSubsonicRejectsChangedSongOrAlbum)$' -count=1`
  - `go test ./internal/db/sqlite -run '^(TestSubsonicAlbumDivergenceDoesNotRetry|TestSubsonicDivergenceClearsPreviousRetry)$' -count=1`
- A guarda de prévia e o índice de `metadata_retries` seguem fora do escopo desta rodada, conforme instrução.
