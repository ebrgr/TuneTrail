# Issue 012 — importação e exportação por perfil

**Atualização na issue 015:** integração Go, jornadas reais e comparação visual
registradas em [ISSUE_015.md](ISSUE_015.md). As notas abaixo descrevem a entrega
anterior à integração; o aceite final do usuário permanece pendente.

Núcleo implementado em 05/11/2026. [Issue](<../../tasks/iss-012 - 012-—-Migrar-importação-e-exportação-por-perfil-para-Go.md>) · [Plano](<../../docs/roadmaps/doc-007 - Roadmap-4-—-administração-importação-chaves-edição-deleção-e-mesclagem.md>).

O handler Go isolado oferece importação e exportação do perfil selecionado. O
formulário identifica nome, slug e ID, exige confirmação e valida sessão, origem
e CSRF antes do staging multipart. O perfil é escolhido pela rota; o usuário
registrado no arquivo não altera o destino. Download é administrativo, no-store e
em anexo. A integração no dispatcher e o aceite visual ficam para a 015.

A importação permanece síncrona. O serviço usa `RowsAffected` para diferenciar
escutas inseridas de duplicadas; entradas ignoradas e erros fatais são contados
separadamente. Em falha ou cancelamento, o prefixo já salvo e eventuais metadados
permanecem; o relatório só é disponível enquanto o cliente estiver conectado.
Uploads são staged fora da árvore de importação do boot, com nomes gerados e
permissões restritas; falha de arquivamento é reportada, não apagada como sucesso.
A aplicação não retoma uploads HTTP após reinício.

Limites independentes: corpo HTTP de 100 MiB com multipart; JSON de 32 MiB e até
500 mil tokens estruturais antes da desserialização; até 250 mil itens/linhas;
ZIP de até 1000 membros, 256 MiB expandidos e linha JSONL de 1 MiB. O limite de
500 mil tokens pode restringir arquivos válidos muito antes dos 250 mil itens.
A janela de importação é preservada. Esperas e acesso serializado ao catálogo
respondem a cancelamento; Koito e ingestão compartilham a serialização.

A exportação usa cursor por timestamp e faixa, incluindo datas anteriores a 1970,
e verifica erros de escrita. A leitura é dinâmica, **não é snapshot**: uma
inserção retroativa pode ficar antes do cursor, metadados/exclusões podem mudar
entre páginas e ingestão contínua pode estender o trabalho. Após início do stream,
erros são registrados e a conexão termina sem anexar JSON de erro ao arquivo.

## Verificação

- Suíte Go completa e testes de concorrência dos pacotes modificados passaram.
- Fixtures fictícias dos cinco formatos, contagem de duplicatas em repetição,
  isolamento entre perfis e erro de JSON; ZIP vazio e linha excessiva; JSON denso
  rejeitado antes de alocação; cancelamento e falha de arquivamento com prefixo salvo.
- Fluxo HTML isolado: sessão, CSRF, importação, resultado, autorização e download.
- Exportação com 1012 escutas, mais de uma página, timestamps iguais e anteriores
  a 1970; writer que falha e Unicode no nome do perfil.
- Revisão Advisor (Sol): preflight de JSON, associação concorrente no catálogo,
  classificação de erros e validação ListenBrainz corrigidos; parecer aprovado.

Sem contato com serviços musicais externos ou uso de dados pessoais nos testes
novos. Screenshots e comparação visual foram adiados para o final da 015.

Na API de arquivo manual, a entrada selecionada é removida da pasta observada no
boot antes do processamento. Em erro, fica em `import_failed/{perfil}`; se não
for possível movê-la, permanece em `import_staging` e a resposta informa isso.
Um teste verifica a ausência na fila e a preservação para recuperação.

O aceite abrangente permanece para a integração: ainda faltam reproduções
específicas de desconexão durante trabalho remoto, falha ao gravar a quarentena,
reinício imediatamente após encerramento abrupto e round-trip completo em outro
perfil sob mutações concorrentes. A execução real de importações não é atômica.
