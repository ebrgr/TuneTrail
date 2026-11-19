---
id: doc-041
title: >-
  Aplicativo multiusuário: contas, administrador e visibilidade das escutas
  (ISSUE-100)
type: specification
created_date: '2026-11-17T17:17:00Z'
updated_date: '2026-11-19T15:26:00Z'
tags:
  - design
---
# Aplicativo multiusuário — contrato da ISSUE-100

## Estado e precedência

Especificação para revisão, associada à decision-010 **proposed**. Consolida as decisões do usuário registradas na ISSUE-100 em 17/11/2026 e substitui as propostas incompatíveis do primeiro parecer do Advisor. Não representa implementação, aceite da decision-010 ou autorização de publicação.

São normativos os requisitos abaixo. A única pergunta em aberto é a permanência do limitador compartilhado de login já existente (§9). Essa pendência não autoriza criar outro limitador. As tarefas de implementação do §10 são propostas, não foram criadas.

Referências: doc-021/doc-022 (modelo anterior e migração v3), doc-010 §7 (escopo anterior), doc-039/decision-008 (identidade musical), doc-040/decision-009 (ingestão), ISSUE-080/133 (schema), ISSUE-123/144/141/074/125–128/153/157 (interface e gestão). Os contratos de identidade musical, retenção e deduplicação das decisões 008/009 permanecem válidos; suas referências à autorização administrativa devem passar a distinguir dono do perfil e administrador conforme este contrato.

## 1. Identidade e dados

Cada conta autentica **exatamente um perfil**, e cada perfil pertence a exatamente uma conta. Não há senha separada de visualização, pertencimento N:N, uma conta dona de vários perfis ou senha root compartilhada.

Manter users e listening_profiles separados. Acrescentar a listening_profiles:
- user_id INTEGER NOT NULL UNIQUE REFERENCES users(id) ON DELETE RESTRICT;
- require_login INTEGER NOT NULL DEFAULT 0 CHECK (require_login IN (0,1)).

users conserva username, hash bcrypt e role user/admin. O login pode diferir do slug público; nome de exibição e slug não são credenciais. A FK e UNIQUE impedem perfil sem dono e dois perfis para a mesma conta; criar/excluir a dupla em transação e validar a existência de um perfil para toda conta completam a cardinalidade 1:1.

sessions.user_id continua identificando o ator autenticado. listens.profile_id e api_keys.profile_id continuam identificando o histórico e a ingestão. user_id legado nessas tabelas e legacy_user_profiles não conferem autorização. Não mover escutas para users, renumerar IDs ou reatribuir chaves ao mudar o papel.

O checkbox **Administrador (root)** representa users.role=admin. Qualquer conta pode receber esse papel, exclusivamente por ação de um administrador. Root significa administrador do aplicativo; não é conta do sistema operacional nem uma credencial adicional.

## 2. Permissões

Dono significa listening_profiles.user_id igual ao ID da conta autenticada. Entrar como usuário comum não libera escutas protegidas de outro usuário.

| Ação | Visitante | Usuário comum | Administrador (root) |
| --- | --- | --- | --- |
| Diretório /, nome, slug, avatar e indicação de proteção | Todos | Todos | Todos |
| Leitura de escutas públicas e seus derivados | Sim | Sim | Sim |
| Leitura de escutas protegidas e seus derivados | Não | Somente seu perfil | Todos |
| Adicionar, editar ou excluir escutas | Não | Somente seu perfil | Todos |
| Importar/exportar histórico; criar, consultar e revogar chaves | Não | Somente seu perfil | Todos |
| Nome, slug, avatar e checkbox require_login | Não | Somente seu perfil | Todos |
| Cadastro/exclusão de conta e perfil | Não | Não | Sim |
| Atribuição/revogação do papel admin | Não | Não | Sim |
| Troca de senha própria | Não | Sim | Sim |
| Emissão de token de recuperação para outra conta | Não | Não | Sim |
| Configuração e operação global do servidor | Não | Não | Sim |
| Capas, aliases, merges e outras alterações do catálogo compartilhado | Não | Não | Sim |
| Configuração de relay do perfil | Não | Somente seu perfil | Todos |

Toda gestão humana exige sessão válida e autorização **no servidor**, inclusive chamadas diretas aos serviços. Ocultar botões não é controle de acesso. Corpo, query, slug, ID de chave ou seleção de perfil não podem substituir o dono/escopo resolvido pelo servidor. Usuário comum não pode atribuir role nem alterar o vínculo user_id pelo formulário ou API.

O último administrador não pode ser excluído nem rebaixado. A consulta do número de administradores e a alteração devem ocorrer na mesma transação de escrita, inclusive em requisições concorrentes; uma validação apenas no navegador ou mutex do processo não basta. Se existir desativação de contas, ela também deve preservar ao menos um administrador apto a entrar.

Excluir conta/perfil é operação conjunta e confirmada: revogar sessões e tokens, excluir os dados pessoais vinculados, chaves, estado temporário e relay do perfil, preservando contas, escutas e catálogo de outros perfis. A exclusão da última conta é impedida pela regra do último admin. Edição/exclusão de escutas próprias não deve alterar ou apagar implicitamente entidades compartilhadas. Capas, aliases e merges globais continuam exclusivos do root; não criar sobreposição de catálogo por usuário nesta entrega.

O administrador não pode consultar senha nem hash em páginas, API ou logs. Pode redefinir o acesso por token, sem recuperar a senha anterior.

### Exceção de importação do dono — decisão do usuário em 19/11/2026

O usuário reafirmou expressamente a **opção 2**: a importação pelo usuário comum deve ser mantida; limitar demais tiraria funcionalidade do não-root. O dono pode importar seu histórico e criar entidades musicais novas com nomes, aliases e imagens do arquivo. Isso é uma exceção documentada à linha de catálogo da matriz, não autorização para editar entidades existentes, merges ou operações editoriais compartilhadas. A confirmação está registrada na ISSUE-160 com autoria `ebrgr`; não equivale a aceite da implementação ou publicação.

Há dois riscos conhecidos nessa exceção:

1. **SSRF cego por image_url:** uma URL do arquivo poderia provocar GET do servidor à rede interna quando um visitante solicita a imagem. A mitigação preserva a imagem e marca a origem não-root no valor persistido de image_source, com o fragmento reservado `#tunetrail-public-only`, que também permanece no export/reimport e nunca é enviado ao servidor HTTP. A busca dessas fontes aceita somente HTTP(S), sem credenciais embutidas, resolve DNS, rejeita respostas privadas/locais, conecta ao IP literal validado e confere o endereço efetivamente conectado. Bloqueia loopback, endereços não especificados, link-local, RFC1918, ULA, metadata de nuvem e equivalentes IPv6/mapeamentos de transição. A restrição vale em cada redirect, sem proxy de ambiente, com até três redirects seguidos, deadline total de dez segundos e corpo de até 10 MiB. Fontes do root sem essa marca mantêm suporte a servidores legítimos em rede local. Não aplicar um bloqueio global às fontes do root por consequência desta exceção.
2. **Pré-semeadura do catálogo por MBID/aliases:** metadados escolhidos no arquivo poderiam capturar a resolução futura de outros perfis. A mitigação usa o campo source já existente: todos os aliases de entidades criadas por import não-root recebem `ProfileImport:<profile_id>` e só participam da resolução automática de nomes daquele destino. Continuam disponíveis para exibir o histórico e exportá-lo; consultas por ID e a administração root continuam possíveis. MBIDs desconhecidos do arquivo não são atribuídos às entidades novas; MBIDs já existentes podem resolver uma entidade do catálogo sem editá-la. Enriquecimento automático posterior não promove MBIDs nem aliases dessas entidades ao namespace global; uma validação editorial explícita, autorizada ao root, pode fazê-lo. A resolução mantém a identidade do destino mesmo ao remover o escopo de leitura de escutas. Perfis usam IDs persistentes que não são reutilizados por cadastro normal. Não foi necessária nova migração além da 006.

A consequência funcional é poder haver entidades com o mesmo nome em históricos de destinos distintos até reconciliação explícita pelo root. Não se exige remover imagens ou aliases do histórico do dono. Os metadados de arquivos continuam sem certificação semântica de autenticidade; as medidas acima limitam o efeito da importação sobre a rede e sobre a resolução de outros perfis. O contrato de ingestão ListenBrainz e seus clientes permanece como está.

## 3. Visibilidade

O checkbox **Exigir login para ver as escutas** corresponde a require_login:
- desmarcado: leitura pública;
- marcado: leitura somente pelo dono ou por qualquer administrador autenticado.

**Todos os perfis, existentes e novos, são públicos por padrão.** A proteção é uma alteração posterior feita pelo dono ou administrador. A raiz / lista todos os perfis disponíveis com nome, slug, avatar e cadeado quando protegido. Não ocultar o perfil do diretório nem confundir a opção com anonimato. Não publicar username de login por consequência de listar o perfil.

Aplicar a política central antes de consultar/serializar escutas em páginas, API, busca, histórico, estatísticas, contagens, rankings, retrospectiva, now-playing, fragmentos e detalhes de mídia. Endpoints legados sem slug também devem passar pelo gate, sem fallback para dados globais ou outro perfil.

A página de perfil protegido sem sessão deve oferecer login sem amostra das escutas. A API deve negar a leitura sem devolver dados pessoais; para usuário comum de outro perfil, negar sem revelar contagens ou conteúdo. Metadados públicos de diretório podem continuar acessíveis. Não vazar escutas protegidas em títulos, OpenGraph, atributos HTML, JSON embutido, sugestões, erros ou cursores. O diretório não pode mostrar última atividade, contagens ou capas derivadas do histórico protegido.

Widget/SVG, cards, RSS e SSE, quando disponíveis ou implementados, devem obedecer à mesma política. Não criar token público de leitura nem reutilizar a chave de ingestão como senha de widget. Exportação exige sessão e permissão mesmo quando as escutas são públicas.

### Imagens e cache

Avatar de perfil permanece público por decisão do diretório. Imagens e URLs derivadas de escutas protegidas não podem contornar o gate. A rota global de imagens deve verificar a elegibilidade antes de servir cache, variantes ou redownload; UUID não é autorização. Uma capa compartilhada legitimamente pública pode permanecer acessível, sem revelar que determinado perfil protegido a ouviu. Não permitir enumeração de catálogo exclusivo do histórico protegido.

Respostas protegidas ou personalizadas devem usar cache privado sem armazenamento compartilhado; autorizar antes de retornar 304. Alternar require_login deve invalidar respostas/fragmentos/widgets e referências de cache que poderiam continuar expondo escutas. Não deixar conteúdo protegido em cache público/CDN nem em URLs de exportação acessíveis sem sessão. Não é possível retirar cópias anteriormente baixadas ou publicadas por terceiros.

### Cache legado — decisão do usuário em 19/11/2026

Decisão do usuário em 19/11/2026 — opção 1. Respostas legadas de /image/{image_id}/{filename} com Cache-Control public, max-age=2592000 são cópias já distribuídas. A origem autoriza todas as requisições antes de servir bytes ou 304. Imagens de perfil protegido usam private, no-store. Alternar require_login muda o validador da imagem. O usuário aceita que caches fora do controle do servidor possam reutilizar os cabeçalhos antigos por até 30 dias. A decisão dispensa corte operacional e purga de caches compartilhados.

### Relay e integrações: consequência aceita

**Marcar require_login NÃO pausa nem altera a ingestão ou o relay de saída.** Recebimento por chave de máquina e encaminhamento aos destinos configurados continuam ativos, sem consentimento adicional ou suspensão automática por privacidade.

Escutas protegidas localmente podem continuar sendo publicadas e lidas no destino externo. Essa consequência foi expressamente aceita pelo usuário e deve constar da descrição da opção de proteção/relay. require_login protege a leitura local, não é promessa de sigilo no destino. Exclusão local não desfaz publicação remota. Preservar a elegibilidade de eventos e os contratos de relay/outbox das decisões 008/009; este contrato não autoriza encaminhar pendências indiscriminadamente.

## 4. Ingestão por chave de máquina

Manter a API ListenBrainz e as chaves atuais: token resolve api_keys.profile_id, e o payload não escolhe outro destino. Chave continua aceitando ingestão/playing-now e validação compatíveis, sem conferir sessão humana, leitura privada, exportação, alteração de papel ou administração.

Criar/revogar chave exige dono autenticado ou admin. Chaves não são transferidas entre perfis. Revogação bloqueia novos envios e preserva o histórico. Não trocar tokens existentes, impor rotação nem alterar o modelo de armazenamento/revelação de chaves nesta entrega.

## 5. Autenticação e recuperação

### Padrão atual preservado

Manter bcrypt e a validação de senha existentes. Não migrar para Argon2id, alterar custo do bcrypt nesta tarefa, impor mínimo de 15 caracteres, lista de senhas comuns, expiração periódica ou nova política de composição.

Referência do comportamento atual: SaveUser/UpdateUser usam bcrypt.DefaultCost; validação de armazenamento tem mínimo de oito caracteres e normalização/limites existentes. ChangeAccount verifica mínimo de oito caracteres e limite de 72 bytes. Essas diferenças existentes não autorizam uma reforma da política na ISSUE-100. O fluxo novo deve reaproveitar os validadores pertinentes; recuperação usa o caminho de troca de senha existente e cadastro usa o caminho de criação existente.

Ampliar login e autenticação de sessão para role=user e role=admin, separando autenticar de exigir admin. Manter mensagens sem exposição de senha/hash. Não introduzir reautenticação recente, MFA, limite de sessões ou outros controles adicionais nesta entrega.

Não criar limite de tentativas, bloqueio, atraso progressivo, bucket por usuário ou lógica por origem/IP. Tentativas por usuário permanecem ilimitadas. A única pendência é o limitador compartilhado já existente (§9), sem ampliá-lo.

### Sessões e formulários

Reutilizar sessões por user_id, duração atual (24h ou 30 dias com lembrar), cookie tunetrail_session HttpOnly e seus atributos atuais; não reformular o formato de token/cookie nem adicionar nova expiração. Múltiplas sessões continuam possíveis.

Preservar a revogação transacional de todas as sessões na troca de senha já existente, inclusive no reset. Preservar a proteção de SaveAuthenticatedSession contra login concorrente autenticado com senha antiga, removendo apenas a exclusividade role=admin. Resolver o papel atual no servidor em cada ação impede uma sessão de conservar acesso root depois de rebaixamento.

Manter CSRF, SameOrigin, preauth de login, ParseSessionForm, limites de parsing/upload e tickets de uso único existentes. Generalizar a vinculação de CSRF para usuário autenticado, sem tratá-lo como admin. Ticket não substitui autorização; conferir sessão/perfil/ação no servidor e revalidar ao aplicar uma operação revisada. Não introduzir novos controles de autenticação sob pretexto de adaptar CSRF.

### Recuperação sem e-mail

Administrador emite token temporário de uso único para a conta escolhida. O destinatário define sua senha pelo fluxo de redefinição, sem receber senha permanente escolhida pelo admin. Usar token aleatório, guardar seu digest, registrar validade (proposta operacional: 30 minutos para recuperação), invalidar o anterior ao emitir outro e não expor o segredo em logs ou respostas públicas.

A posse do token só autoriza definir a senha daquela conta; não concede sessão de leitura/gestão. Validar validade/uso e consumir o token junto à atualização do hash e revogação das sessões em transação. Depois solicitar login normal. Não é necessário serviço de e-mail. Ativação de conta nova ou migrada pode usar o mesmo mecanismo, emitido sob demanda pelo admin; não compartilhar senha entre perfis.

Para perda de acesso do único admin, prever recuperação local pelo operador com acesso ao servidor, sem senha mestra ou endpoint remoto de elevação. Isso não permite cadastro público nem concessão automática de admin.

## 6. Bootstrap e migração

Não editar migrações já aplicadas. A ISSUE-080 planeja v4 de ingestão e a ISSUE-133 coordena a migração seguinte de identidade musical (v5 é sugestão, não reserva). Definir a versão desta migração **em conjunto com 080/133 antes de escrever SQL**; não reservar nem ocupar v4/v5 em paralelo. Atualizar schemaVersion, validação de layout, bootstrap e fixtures, com recusa de versão desconhecida. Esta decisão não exige implementar toda a ingestão para começar os serviços de contas.

Procedimento obrigatório:
1. Backup consistente incluindo WAL e ensaio de restauração; inventariar usuários, pontes legadas, perfis, escutas e chaves por perfil.
2. Reutilizar a conta administrativa atual, seu hash bcrypt e o vínculo de perfil comprovado na ponte legada; preservar o papel. Não associar contas por coincidência de nome nem promover usuários comuns automaticamente.
3. Criar um usuário para cada perfil sem conta, com username único sugerido pelo slug e colisões resolvidas explicitamente. Entregar acesso por token de ativação emitido pelo admin, não por senha temporária compartilhada ou previsível. Contas aguardando ativação não podem autenticar com senha vazia; representar esse estado explicitamente. Enquanto aguardam, a leitura pública e a ingestão seguem o contrato e o admin pode gerenciá-las.
4. Contas legadas sem perfil devem ganhar seu único perfil explicitamente, sem apagar ou juntar dados; inconsistências de mapeamento exigem reconciliação antes de concluir. Remover o trigger trg_legacy_user_profile na nova migração antes das inserções para não duplicar perfis e substituir a ponte pelo vínculo canônico.
5. Aplicar user_id NOT NULL UNIQUE e require_login=0 a todos os perfis existentes. Novos perfis também usam 0 por padrão. Não alterar IDs/slugs, timestamps, histórico, chaves/tokens, vínculos de relay ou destino de arquivos por perfil. A conta admin mantém a senha; não impor troca por esta migração.
6. Cadastro novo deve criar conta+perfil+chave inicial em uma transação. Instalação vazia cria o primeiro admin com seu único perfil pelo bootstrap existente adaptado. Não recriar admin automaticamente numa base com contas e sem administrador: exigir recuperação local.
7. Verificar FK/integridade, cardinalidade 1:1, ao menos um admin e contagens/associações de escutas e chaves antes/depois. Testar falha/rollback e reinício sem duplicações. Preservar sessões existentes quando o vínculo continuar válido; a ampliação para user não exige invalidá-las em massa.
8. Não reconstruir a identidade musical nesta migração: manter a PK atual de listens ou os event_id já introduzidos pela 133, conforme a ordem coordenada. Restauração de backup deve observar reconciliação de workers/relay conforme decision-009.

## 7. Telas e fronteiras de implementação

Login serve a todas as contas. Separar conta autenticada, perfil visitado e permissões da ação; o booleano adminSession/publicProfile atual não basta para liberar edição própria.

- ISSUE-125: conta de qualquer usuário, alteração de login/senha e saída; não chamar todo usuário de administrador.
- ISSUE-126: gestão root de contas e perfis juntos, ativação/reset, cadastro/exclusão e checkbox Administrador (root); edição própria e checkbox Exigir login para ver as escutas, sem oferecer alteração de papel ao comum.
- ISSUE-127: configuração de servidor continua exclusiva root.
- ISSUE-128: login comum, revisões e mensagens distinguindo manejo próprio e global.
- ISSUE-157: confirmação GET e logout POST protegido também para usuário comum; manter revogação só da sessão atual e funcionamento sem JavaScript.
- ISSUE-153: preferências globais em conf.toml continuam editáveis somente pelo root; não inventar preferência por usuário.
- ISSUE-074 e navegação atual da 156: preservar padrão responsivo, traduções e aprimoramento progressivo. ISSUE-123 é base histórica, não restaurar a segunda barra removida.
- ISSUE-141/144: gestão de escutas do dono, revisão de exclusão e estados vazios devem respeitar o escopo; operações globais de catálogo continuam root.

Áreas: internal/admin, internal/adminui, internal/adminweb, internal/webui, engine/handlers/auth.go e handlers de perfil/escutas/import/export/chaves/imagens, engine/middleware, engine/routes.go, bootstrap, models e store SQLite. A política deve ser compartilhada entre HTML, JSON e serviços. Fluxo de leitura: sessão opcional → perfil → permissão de leitura → consulta com profile_id. Fluxo de gestão: sessão obrigatória → CSRF/ticket aplicáveis → permissão por ação/perfil → gravação com verificação transacional quando necessária.

## 8. Testes de aceite da implementação futura

Esta entrega documental não executa nem comprova estes testes:
- Dois usuários comuns entram separadamente; cada um gerencia só seu perfil. Um perfil público permite leitura sem login e continua negando toda gestão ao visitante.
- Root gerencia todos e configura o servidor; qualquer usuário pode ser promovido/rebaixado pelo root, comum não se promove por API/formulário e perde acesso global após rebaixamento.
- Concorrência entre exclusões/rebaixamentos nunca remove o último admin.
- Raiz lista todos, inclusive protegidos, com nome/avatar e sem dados de escuta protegida. Login no usuário A não libera escutas protegidas de B.
- Gate cobre páginas, API, busca, mídia, fragmentos, now-playing e widget disponível; imagens, cache e 304 não burlam proteção. RSS/SSE futuros devem cumprir o mesmo contrato antes de existir acesso público.
- Ingestão e relay de entrada/saída continuam ativos ao marcar proteção; chaves atuais continuam válidas e não permitem gestão ou leitura privada.
- Login/cadastro/troca/reset mantêm bcrypt e validadores existentes; não há Argon2id, novos mínimos, bloqueio ou quota por usuário/IP. Teste do limitador existente segue a resposta do §9.
- CSRF, parsing e ticket replay continuam protegidos; troca/reset revogam sessões e login concorrente com senha anterior não as recria. Token expirado/usado ou de outra conta não redefine senha nem dá acesso ao histórico.
- Migração de banco novo, v3 com um/vários perfis e esquema coordenado 080/133 preserva contas, hashes, escutas e chaves; perfis são públicos e relação 1:1 íntegra. Backup/restore, rollback e reinício comprovados em cópias locais.
- Usuário comum não altera capas/aliases/merges; importação/exportação e exclusão por IDs manipulados não alcançam outro perfil.
- Fluxos sem JavaScript e capturas 390/768/1440 nos temas Yuu/Pearl conforme 074, com aceite visual posterior.

## 9. Única pergunta pendente

**Decisão do usuário (2026-11-17): MANTER COMO ESTÁ, sem complicar as senhas; nenhum limite novo.** Pergunta original: manter como está ou remover o limitador compartilhado de login já existente (10/min quando habilitado, engine/routes.go)? Recomendo manter como está**, para preservar o comportamento atual, sem criar limites por usuário, bloqueios ou análise de origem/IP. A escolha não foi tomada nesta especificação.

“Como está” inclui exatamente o comportamento de agrupamento/configuração do middleware atual; não o reimplementar como bucket por usuário ou IP. Se a intenção de não considerar origem/IP também exigir eliminar qualquer comportamento de agrupamento já fornecido por esse middleware, a opção coerente é removê-lo, não substituí-lo por outro. A recomendação de preservá-lo não autoriza acrescentar lógica de origem.

O AC2 atual da ISSUE-100 ainda cita “limite de tentativas”. Esse trecho precisa ser alinhado à resposta acima antes da implementação; não é autorização para novos limites e não foi marcado como verificado.

## 10. Issues de implementação propostas (não criadas)

IDs abaixo são referências locais P1–P7 para planejar dependências, não números ISSUE reservados. Atribuir ISSUE-NNN apenas quando o usuário autorizar a criação. Todas derivam da ISSUE-100 e dependem do aceite da decision-010; não há aceite implícito pela entrega do documento.

| Ordem | Título proposto | Tipo | Labels | Dependências | ACs essenciais |
| --- | --- | --- | --- | --- | --- |
| P1 | Schema e migração de contas e perfis 1:1 | feature | banco, admin | ISSUE-100; coordenação de versão com ISSUE-080/133 | FK NOT NULL UNIQUE; require_login=0 novos/legados; vínculo admin/hash/IDs/escutas/chaves preservados; trigger legado removido; bootstrap atômico; rollback/restore e cardinalidade testados |
| P2 | Autenticação e sessões para todos os usuários | feature | admin, api | P1; resposta ao §9 | Login user/admin com bcrypt/validação atuais; separar sessão de papel; CSRF/tickets adaptados; troca revoga sessões e corrida com login antigo coberta; sem novos limites/bloqueios por usuário/IP |
| P3 | Autorização de gestão própria e administração global | feature | admin, api | P1, P2 | Dono só maneja seus dados/chaves/import/export; root todos e servidor; catálogo compartilhado só root; role atual no servidor; último admin protegido em transação e teste concorrente |
| P4 | Gate de visibilidade das escutas, imagens e cache | feature | api, ui, imagens | P2, P3 | Diretório com todos; checkbox gate uniforme; nenhum dado protegido em busca/widget/imagens/cache/304; público ainda exige login para gestão; ingestão e relay inalterados testados |
| P5 | Gestão root de contas e perfis com ativação e recuperação por token | feature | admin, banco, api | P1, P2, P3 | Cadastro/exclusão só root e atômicos; qualquer perfil promovível; token temporário único sem e-mail; reset revoga sessões; ativação sem senha compartilhada; não excluir último admin; testes de token expirado/reutilizado |
| P6 | Adequar telas multiusuário e navegação por permissão | enhancement | ui, admin, i18n | P3, P4, P5; coordenar ISSUE-125/126/127/128/157/153/074 | Login/conta para comum; checks de papel/proteção; gestão própria e root distintas; aviso relay; logout protegido comum; semJS e capturas/aceite responsivo |
| P7 | Validar o corte multiusuário e documentar operação | chore | validacao, docs, operacao | P1–P6 | Matriz HTML/JSON/serviços; banco migrado sem perda; restore e regressão de ingestão/relay; make check; evidências sem segredos; docs e CHANGELOG da implementação |

P6 deve preferencialmente ser distribuída entre as ISSUE-125–128 e 157 já existentes, reescopando-as em vez de criar duplicatas; o título acima identifica o pacote de trabalho. ISSUE-153 mantém seu escopo global. A ordem de P1 é coordenada com 080/133, sem prometer número v6 ou impor conclusão de todo o pipeline de ingestão. Não liberar uma versão intermediária que permita login comum e exponha gestão global ou escutas protegidas por ausência de P3/P4.
