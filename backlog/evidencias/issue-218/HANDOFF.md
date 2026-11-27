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
