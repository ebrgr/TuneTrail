# Issue 040 — persistência de configuração

Implementação em 09/11/2026. [Issue](<../../tasks/iss-040 - 040-—-Persistência-e-concorrência-da-configuração.md>) ·
[Contrato](<../../docs/design/doc-019 - Issue-038-—-contrato-e-plano-da-configuração-administrativa.md>). A 039 foi encerrada por
solicitação do usuário; esta issue foi fechada por aceite do usuário em 09/11/2026.

## Entrega

O pacote `internal/cfg` oferece o serviço `SettingsStore` para a futura 041:

- `NewSettingsStore(getenv)` fixa o caminho e os valores ativos obtidos no startup.
  O chamador da aplicação deve fornecer `os.Getenv`, não valores de formulário.
- `Read(ctx)` retorna revisão, existência, dez valores salvos/efetivos/ativos,
  origens do arquivo e efetivas, campos sobrescritos e indicação de reinício.
- `Save(ctx, revision, updates)` aceita somente as dez constantes `KOITO_*` do
  contrato, com valores comuns visíveis e até 64 KiB no conjunto. Valida TOML com
  defaults e composição efetiva, sem incorporar overrides ao arquivo. Não altera
  a configuração ativa, banco, variáveis de ambiente ou relay.
- `ErrSettingsConflict` indica revisão obsoleta; a futura interface deve traduzi-lo
  em HTTP 409. `ErrInvalidSettings`, `ErrUnsupportedSettingsFormat`,
  `ErrSettingsReadOnly` e `ErrSettingsPath` distinguem outras falhas.
- `ErrSettingsDurability`/`DurabilityError` significam que o rename aconteceu,
  mas o sync do diretório falhou. O retorno inclui estado observado quando possível;
  a interface não deve informar que a gravação foi desfeita.

O serviço é interno e não cria endpoint nem tela nesta issue. A edição
administrativa autenticada, a apresentação dos campos e o mapeamento HTTP são
trabalho da [041](<../../tasks/iss-041 - 041-—-Edição-de-configuração-pelo-master.md>).

## Preservação e concorrência

O editor altera intervalos de valores escalares, preservando os demais bytes,
comentários, formatação e finais de linha; novos campos são inseridos nas tabelas
corretas. Após a edição, o arquivo é novamente decodificado e todos os campos são
comparados para detectar mudança semântica fora do patch.

São editáveis os documentos com tabelas/chaves simples e valores escalares em
uma linha. TOML válido com chaves quoted/dotted, tabelas inline ou strings
multilinha é recusado com `ErrUnsupportedSettingsFormat`, sem modificar o arquivo;
o leitor da 039 continua aceitando formatos TOML válidos. Campos e versões
não reconhecidos também não são sobrescritos silenciosamente.

Revisão SHA-256 distingue arquivo ausente de arquivo presente. Escritas usam gate
no processo e `conf.toml.lock` com `flock` exclusivo. Editores externos cooperativos
devem bloquear esse mesmo arquivo e não removê-lo/recriá-lo durante uso. Leitura
usa lock compartilhado quando ele existe, sem precisar criá-lo em volume somente
leitura. A escrita relê o documento e repete a revisão logo antes do rename;
alterações externas não cooperativas na janela final não são universalmente
impedidas. Na escrita, links e arquivos especiais no destino ou no lock são recusados.

O temporário fica no mesmo diretório, é sincronizado antes do rename e removido em
falhas anteriores à publicação. Arquivo novo usa modo 0600; quando possível,
arquivo existente preserva modo e proprietário/grupo, ou a operação falha antes
de publicá-lo. Depois do rename, sincroniza-se o diretório. Leitura do TOML aceita
permissões somente leitura, mas o servidor/SQLite ainda exige dados graváveis.

## Validação executada

- `CGO_ENABLED=0 go test -count=1 ./...` e `go vet ./...`: passaram.
- `CGO_ENABLED=1 go test -race -count=1 ./internal/cfg`: passou.
- Builds Linux estáticos sem CGO, amd64 e arm64: passaram; não houve execução
  dos binários Linux nesta verificação.
- `validate_multiuser.py` com binário e bases temporárias: jornadas de banco novo
  e migrado v2 passaram, com integridade SQLite confirmada.
- Testes dirigidos cobrem dez campos, ambiente/_FILE, salvo/ativo/override,
  diretório fixado no startup, comentários, CRLF, inserções, formatos não
  suportados, validação das duas composições e restauração de configuração válida.
- Concorrência de oito salvamentos, alterações externas antes do rename, lock
  cooperativo e cancelamento, arquivos somente leitura, links, FIFOs e limites de
  tamanho foram exercitados. Falhas injetadas em sync de arquivo e rename
  preservaram o original; falha de sync de diretório retornou estado publicado e
  durabilidade incerta. Arquivos temporários foram verificados após falha.
- Advisor/Sol revisou arquitetura e implementação. Achados de inserção no EOF,
  correção de override, normalização de inteiro e FIFOs foram corrigidos e
  cobertos por regressões; parecer final aprovado. O Advisor não executou testes.
- Graphify externo atualizado após as mudanças.

Os testes de falha são injeções locais, não simulação exaustiva de falta física
de energia ou garantia para sistemas de arquivos de rede. Nenhuma base operacional
foi usada. Capturas e validação visual pertencem à 041/042.
