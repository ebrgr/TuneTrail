ROADMAP.md

## TAREFAS

1. Listar todas as variavies de ambiente que são passadas e podem ser usadas. Após começar a estudar a migração de todos estes valores para um arquivo `conf.toml` por examplo que mora no mesmo diretório do banco de dados.
   - Após fazer com que todos estes dados sejam possíveis de serem editados com a senha e usuário master
   - [Estudo, inventário e issues 038–042](ROADMAP_02_CONFIGURACAO_TOML.md) — 038 retomada: [contrato dos dez campos, fuso, Subsonic e relay](CONTRATO_CONFIGURACAO_038.md) em planejamento; 039 fechada e aprovada; 040 fechada e aprovada; 041 fechada e aprovada; 042 implementada e aguardando aceite.

2. Melhorar o layout de adiminstração e separar opções por usuários de forma mais efetiva, senha simples para cada usuário para deixar perfil público/privado
    - Colocar badge do lado do nome do usuário sempre que conta de administração estiver logada somente esta conta pode apagar dados e adicionar / deletar novos perfis.
    - Senha simples em cada perfil somente para deixar as visualizações públicas ou não

3. Desenvolver a interface multilíngue — [estudo e plano](ROADMAP_03_UI_MULTILINGUE.md), com [issues locais 050–054](issues/README.md). Issue 050 fechada e aprovada; 051 fechada e aprovada; 052–054 implementadas e aguardando teste funcional e aceite. Catálogos externos em `langs`.
4. Integrar Escutas do spotify / deezer / tidal ou de outros serviços ao koito.
5.  Gerar imagens que podem ser postadas em FEEDs de midias sociais como instagram
6. Revisar a documentação e preparar o site oficial do TuneTrail: conteúdo, identidade, endereço, hospedagem, build e capturas. [Issue 048 — documentação e site](issues/048-documentacao-site-tunetrail.md).
7. Aproveitar tags, gêneros e datas de lançamento: priorizar informações recebidas na escuta e complementar pelo MusicBrainz quando necessário, com migração segura do SQLite. [Issue 055 — metadados musicais](issues/055-metadados-tags-lancamento.md).
8. Estudar melhorias de interação e funções dinâmicas na interface. [Issue 056 — interações da UI](issues/056-estudo-interacoes-ui-dinamica.md).

## PENDENCIAS

[Revisão das issues e pendências remanescentes](REVISAO_ISSUES_PENDENCIAS.md).
Estado atual: **47 issues fechadas e 10 abertas** — 038 (planejamento retomado), 042 (implementada, aguardando aceite),
048 (documentação e site), [049 (associação residual de artista e álbum)](issues/049-associacao-residual-artista-album.md), 052–054 (interface multilíngue) [055 (metadados musicais)](issues/055-metadados-tags-lancamento.md) [056 (interações da UI)](issues/056-estudo-interacoes-ui-dinamica.md) e [057 (controles da retrospectiva)](issues/057-ajustes-retrospectiva-controles.md). As issues 043–045 foram fechadas por aprovação do usuário.

Melhorias registradas nas issues encerradas:

- **Banco e importação (046, 020/021):** ampliar testes de interrupção, falha de disco, reinício e restauração.
- **Acessibilidade (024/026):** verificar leitor de tela, zoom de 200%, contraste dos temas e fontes.
- **Estatísticas e fusos (018):** revisar a média por período, calendários excepcionais e desempenho com históricos grandes.
- **Docker e relay (034, 020):** estudar execução sem root e fila persistente para entrega confiável das escutas.
- **Imagens e gráficos (035, 047):** implementar exportação social (tarefa 5) e estudar curvas mais redondas sem ultrapassar os pontos.
- **Registros do projeto:** reconciliar referências históricas a React e os checkboxes da 046 com as evidências existentes.

Os detalhes, prioridades, limitações aceitas e pontos já resolvidos estão no
documento de revisão. As melhorias acima ainda precisam de planejamento próprio.

## CONCLUÍDOS

### Renomear o projeto para outro como TuneTrail. O `icon.png` e `icon.svg`, atualizar a nova identidade visual que já está em `/assets`
   - Como a segunda identidade visual "  TuneTrail
    Your music. Your journey.
    A self-hosted listening history, based on Koito `icone original pequeno` engine version `versão publicada do koito usada como base para o projeto`"
   - Criar um novo Readme.md com o novo logo, trocar o logo do projeto que aparece no browser
   - retire os links do ko-fi
   - retire o link para o demo
   - Enfaze do projeto é a simplificação mudança da arquiterura GO/BAckend-React/Front para monolitico simplificando e retirando dependencias externas, mantendo código puro em GO/HTML/CSS/Minimal JScript
   - [Issue 046](issues/046-migracao-banco-tunetrail-solved.md): backup e migração de `koito.db` para `tunetrail.db`, preservando o esquema v3 — fechada e aprovada.
   - Links e diretório local alinhados ao novo repositório: [issue 045](issues/045-links-diretorio-tunetrail-solved.md) — fechada e aprovada em 09/11/2026.
   - Versionamento próprio TuneTrail 0.x: [issue 044](issues/044-versionamento-tunetrail-solved.md) — fechada e aprovada em 09/11/2026.
   - [Estudo e implementação](IDENTIDADE_TUNETRAIL.md) · [Issue 043](issues/043-identidade-tunetrail-solved.md): identidade aplicada; issue fechada por aprovação do usuário em 09/11/2026, preservadas as limitações de verificação registradas.

### Substituir libvips/bimg por `imaging + webp-go-pure` para executar a aplicação sem CGO e sem bibliotecas nativas de imagens. **Transição concluída nas issues 031–036; geração de imagem social ainda requer planejamento.**
    - Primeiro: migrar capas, avatares e cache, preservando JPEG/PNG de entrada e WebP de saída; validar qualidade, memória e desempenho (issues 031–034).
    - Depois: transformações da retrospectiva preparadas com `imaging`; `x/image` permanece para fontes e texto (issue 035).
    - [Limpeza de executáveis antigos vinculados à libvips](LIMPEZA_LIBVIPS.md).
    - [Plano detalhado e issues](PLANO_TRANSICAO_IMAGENS_GO.md) · [Contrato e medições da 031](comparisons/transicao-imagens/ISSUE_031.md) · [Componente e testes da 032](comparisons/transicao-imagens/ISSUE_032.md) · [Integração e capturas da 033](comparisons/transicao-imagens/ISSUE_033.md) · [Correções da 036](comparisons/transicao-imagens/ISSUE_036.md) · [Distribuição da 034](comparisons/transicao-imagens/ISSUE_034.md) · [Retrospectiva da 035](comparisons/transicao-imagens/ISSUE_035.md) · [Coleta e publicação da 037](comparisons/transicao-imagens/ISSUE_037.md).


### Base de dados vigente: SQLite com esquema consolidado v3.** Referências a v4
são históricas, anteriores à consolidação. Consulte
[Banco de dados](BANCO_DE_DADOS.md) antes de planejar alterações de persistência.

### Estrutura e sequência sugeridas: Transformar aplicação backend/frontend em um serviço monolítico usando HTML, CSS e o mínimo de Javascript, favorecendo apelo visual e rapidez

Preservar banco, migrações, catálogo, importadores e API ListenBrainz. Acrescentar uma camada web com handlers HTML, modelos de apresentação, templates e arquivos estáticos.
Incluindo estás funcionalidades criando novos arquivos linkado ao backend já desenvolvido, para posteriormente podermos usar atualizações do backend do projeto original (upstream) de forma mais "plug and play"

Os handlers HTML devem compartilhar serviços e consultas com a API, sem requisitar a própria API via HTTP.

Sequência de Tarefas Iniciais:

1. Reproduzir layout, fontes, temas, barra lateral e cards. [Análise e plano](ROADMAP_01_INTERFACE.md) · [Prévia Go e screenshots comparativas](comparisons/roadmap-01/README.md) — primeira prévia implementada e validada; revisão completa de paridade pendente.
2. Migrar seleção de perfis, home, rankings e histórico. [Análise e plano](ROADMAP_02_PAGINAS.md) — implementado e validado; issues locais 003–005 fechadas; modo Go opcional. [Comparação e evidências](comparisons/roadmap-02/README.md).
3. Migrar artistas, álbuns, faixas e retrospectiva. [Estudo e plano](ROADMAP_03_MIDIA_RETROSPECTIVA.md) — estudo concluído; núcleos 006–008 e integração 009 implementados; comparações visuais registradas; issues 006–009 encerradas por aceite do usuário em 05/11/2026.
4. Migrar administração, importação, chaves, edição, deleção e mesclagem. [Estudo e plano](ROADMAP_04_ADMINISTRACAO.md) — implementado no modo Go opcional; integração 015 validada, Advisor aprovado e 55 capturas registradas; issues 010–015 encerradas por aceite do usuário em 05/11/2026. [Entrega e comparações](comparisons/roadmap-04/ISSUE_015.md).
5. Validar comportamento, permissões, fusos e isolamento entre perfis. [Estudo e plano](ROADMAP_05_VALIDACAO.md) — estudo com Advisor concluído; issue 016 fechada e resolvida; 017 fechada e resolvida; 018 fechada e resolvida; 019 fechada e resolvida; 020 fechada e resolvida; 021 fechada e resolvida por aceite do usuário. [Evidências iniciais](comparisons/roadmap-05/README.md).
6. Comparar capturas em desktop e celular antes de remover React. [Estudo e plano com Advisor](ROADMAP_06_COMPARACAO_VISUAL.md) — issues 022–025 resolvidas; issue 026 planejada; falha React na issue 028; Go é padrão; React exige seleção explícita.
