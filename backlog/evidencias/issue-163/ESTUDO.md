# ISSUE-163 — Estudo das telas multiusuário

Leitura estática da v0.43 na worktree a partir de `9fdd3e1`. As linhas se referem a essa base. A ISSUE-158 estabelece conta/perfil 1:1 e estados ativo/pendente. As ISSUE-159 a 162 entregam sessão, política central, visibilidade e gestão root com resgate. Os três critérios da 163 continuam abertos. Testes, capturas e aceite visual fazem parte da implementação futura.

## 1. Inventário por papel

«Dono» é a relação entre o ator e o perfil alvo. Um usuário comum é dono do próprio perfil público ou protegido e é terceiro nos demais. Root mantém poderes globais mesmo quando visita um perfil de outra conta.

| Tela ou rota HTML | Hoje | Proposta por papel |
| --- | --- | --- |
| GET `/admin` e POST `/admin/login` | Visitante recebe formulário, opção de lembrar e link de resgate. Login bem-sucedido retorna 303 para `next` validado ou `/admin`. Root chega à administração. Comum ativo sem retorno explícito chega ao 403, qualquer que seja a proteção do seu perfil. | Entrada neutra comum aos dois papéis. Comum segue para Conta ou destino de leitura permitido. Root segue para Administração ou destino de leitura permitido. Sessão existente mantém seu papel e recebe um destino coerente. |
| GET/POST `/admin/redeem` | Tela pública para ativação ou recuperação com usuário, token e senha. A conta pendente só autentica após ativação. Uma emissão root aparece em tela própria, com link de resgate. | Visitante e destinatário veem formulário neutro, erro uniforme e chamada para entrar depois de resgatar. Root emite na gestão da outra conta. O resgate confere apenas poder de definir a senha. |
| Diretório GET `/` | Todos veem nome, slug, avatar e cadeado de cada perfil, sem atividade derivada. O rodapé traz Administração para todos. | Preservar o diretório público. Visitante vê Entrar, autenticados veem Conta e Sair, root também vê Administração. O cadeado anuncia o estado «Protegido». |
| `/<slug>`, escutas, busca, rankings, retrospectiva, mídia e imagens | Visitante e terceiro autenticado leem perfis públicos. Protegido só é legível pelo dono e root. A negativa HTML usa 404 genérico com Entrar inclusive para quem já entrou. A negativa do caminho antigo sem slug com um perfil protegido usa texto puro. | Manter o gate antes de consultar dados. A negativa leva ao diretório e dá acesso a Entrar ao visitante, sem identificar o motivo. Comum autenticado ganha Conta/Sair. Dono e root visualizam o histórico protegido sob cache adequado. |
| Conta GET `/admin?section=account`, POSTs `/admin/account/{username,password}` | O gate e o serviço já aceitam o próprio usuário comum e root. Entrada por URL direta, dentro do template Administração. Há formulário de identidade, senha e botão Sair. | Conta é destino de qualquer sessão, com cartões Identidade, Senha e Sessão. Visitante chega ao login quando usa a entrada de conta. Conta de terceiros, papel e emissão de token ficam fora dessa tela. |
| Gestão GET `/admin?profile=N` e seções `profiles/keys/relay/import/export` com perfil, POSTs `/admin/profile/N/…` e GET `…/export` | Dono já pode abrir pelo endereço e operar no próprio perfil. Root pode escolher qualquer perfil. O formulário inclui nome, slug, avatar e proteção, além de chaves, importação, exportação e relay. Edição de perfil comum usa o mesmo shell administrativo. | Comum encontra «Meu perfil» sem informar ID ou slug. Root usa a lista de contas/perfis e pode abrir os mesmos cartões para o alvo autorizado. Terceiro mantém apenas acesso à leitura pública. |
| Gestão root GET `/admin`, `?section=profiles`, `/admin/profile/N/delete-review`, `/admin/settings`, `/admin/manage/…` | Root cria conta pendente + perfil, atribui papel, emite token para outra conta, revisa exclusão, configura servidor e opera o catálogo. Parte do fluxo de gestão de escutas usa rotas com slug, distintas de operações globais. | Administração só para root. Exibir estado da conta, emissão para terceiros, impacto da exclusão e regras de último admin. Configuração e catálogo globais ficam no grupo root. Dono tem gestão de escutas no seu perfil com ações específicas de escuta. |
| 401, 403, 404 e saída | Login HTML inválido retorna 401 no formulário. `/admin` negado ao comum retorna 403 com POST de logout mínimo, com CSRF e ticket. Outros erros HTML usam o render genérico quando passam por `fail`. O POST de saída já aceita qualquer sessão válida e revoga apenas a atual. | 401 pede login ou nova sessão, 403 indica acesso negado e oferece volta segura e saída, 404 oculta tanto uma rota ausente quanto leitura protegida negada. O GET de confirmação e o POST protegido da 157 atendem aos dois papéis. |
| Navegação lateral/inferior, menu (+), header e avatar | A barra inferior é a versão móvel da navegação lateral. A segunda barra já saiu na 156. No público, o menu trata comum como visitante. Avatar com escudo e atalho de gestão depende de `.Admin`; nas páginas de gestão o sinal deriva de sessão, inclusive comum. A linha de escuta e ações de mídia pública também usam `.Admin`; o dono perde atalhos de escuta autorizados. | Compartilhar destinos entre as duas apresentações. Mostrar o perfil visitado com avatar e link para diretório. Identidade de sessão e permissões são separadas. Permitir «Gerenciar escuta» apenas ao dono do perfil visitado ou root, e ações globais de mídia apenas ao root. |

### Caminhos concretos que levam o usuário comum ao 403

- Login bem-sucedido → `/admin` por padrão em `internal/adminui/admin.go:418–422` → autorização global em `internal/adminui/authorization.go:30` → 403 em `internal/adminui/admin.go:252–254`.
- Diretório → **Administração** em `assets/templates/public/live.html:131`; menu (+) público → **Entrar** em `assets/templates/shared/nav.html:134–140`, pois `internal/webui/handlers.go:70` usa sessão admin como sinal. O link **Entrar** do 404 em `assets/templates/public/live.html:40` tem o mesmo efeito para quem já está autenticado.
- Conta ou perfil próprio por URL direta → **Servidor** no topo de `assets/templates/admin/admin.html:21–22` e no menu `assets/templates/shared/nav.html:128–133` → gate em `internal/adminui/admin.go:228–237`. **Gerenciar perfis** no menu quando o contexto está sem perfil aponta para `/admin?section=profiles` e cai no gate global (`assets/templates/shared/nav.html:122–127`, `internal/adminui/authorization.go:20–30`). Com o perfil próprio, o mesmo link pode funcionar. O erro de apresentação está em `assets/templates/admin/admin.html:14` e `internal/adminui/admin.go:587–588,614`, onde presença de usuário vira `Nav.Admin`. O avatar também pode anunciar indevidamente «Administrador conectado» (`assets/templates/shared/nav.html:84–92`).
- Os links públicos de catálogo estão protegidos pelo sinal admin (`assets/templates/public/media.html:139–147`). Os links de gestão de escuta também, em `assets/templates/public/live.html:361–369`. Neste último caso há uma omissão funcional para o dono, pois o servidor já diferencia gestão da escuta e do catálogo (`internal/adminui/authorization.go:75`).

## 2. Lacunas contra os critérios e débitos herdados

Severidade indica o impacto observado ou o risco que a implementação deve controlar. Os pareceres das ISSUE-159 a 162 registram a evolução das correções.

| Critério ou pendência | Severidade e conclusão |
| --- | --- |
| AC 1, (a) destino e 403 | Alta, usabilidade. Um login válido pode parecer falha. O 403 provisório oferece saída; precisa integrar Conta, diretório e confirmação da 157. Preservar 403 em console explicitamente negado. |
| AC 1, (b) navegação | Alta, usabilidade e fronteira de autorização. Há ações root visíveis para o comum no shell autenticado, embora os gates barrem a execução. Corrigir o modelo de navegação e provar isolamento no HTML e nas chamadas diretas. |
| AC 1, (c) textos e logs | Média, i18n e diagnóstico. `engine/handlers/auth.go:30` já usa “Login failed”. O 401 HTML de credencial inválida está uniforme (`internal/adminui/admin.go:398–410`). “administrator session required” permanece em `engine/handlers/auth.go:80`, num fallback de `MeHandler` cujo chamador aceita `ManageAccount` (`engine/routes.go:113`). A chave também existe em `internal/adminui/messages.go:11`. A falha JSON de logout ainda diz “Administrative logout failed” (`engine/handlers/auth.go:62`). Ajustar os pontos realmente comuns, preservando status e compatibilidade JSON. |
| (d) oráculo de tempo | Média, segurança, regressão. A conferência estática de `internal/admin/auth.go:156–186` confirma uma comparação bcrypt para inexistente, pendente e ativo com senha errada. Inexistente e pendente usam o mesmo hash fictício em `DefaultCost` e não autenticam. Entrada vazia e falha de banco têm fluxos próprios. A equivalência depende de hash ativo válido e custo equivalente, inclusive em credenciais legadas. Reexecutar o teste de instrumentação da 159 na implementação e conferir custo de fixtures migradas, sem presumir igualdade temporal fim a fim ou usar limiar de tempo instável em CI. |
| (e) legado 404 e cache | Média, privacidade/cache e consistência. `internal/webui/routing.go:54–63` retorna 404 textual sem `Vary: Cookie` no caminho negado, embora use `private, no-store`. Alinhar com 404 genérico e política de cache de respostas dependentes da sessão, incluindo o caminho permitido. Não há vazamento confirmado pela leitura. |
| (e) cadeado e evidência | Média, acessibilidade/i18n. O `aria-label` no span genérico em `assets/templates/public/live.html:119` descreve ação e pode não ser anunciado. Ícone decorativo com texto oculto «Protegido». Baixa, rastreabilidade. A ISSUE-161 ainda referencia `REVIEWER.md` com veredito simplificado. Corrigir a referência para o parecer detalhado, preservando o registro histórico. |
| (e) cobertura de regressão | Média, segurança. Cobrir imagem ao root pela URL antiga, PATCH com campos extras (`role`, `user_id`) e igualdade do 404 com perfil inexistente também em rotas como interest, aliases, top/albums e first-activity. Usar o inventário de `engine/visibility_issue161_test.go`. |
| (f) resgate e interface | Média, disponibilidade. O resgate inválido ainda adquire lock de escrita brevemente (`internal/db/sqlite/account_tokens.go:71–76` e verificação da 162). A correção anterior moveu bcrypt para depois da validação do token. Investigar aquisição/latência sob carga em tarefa de backend própria, preservando consumo único, expiração, troca e revogação atômicos. Baixa, risco de regressão. `CreateListeningProfile` e `DeleteListeningProfile` continuam em `ProfileStore` (`internal/db/db.go:133,136`) sem validar sessão, com uso restrito a testes segundo o parecer. Auditar chamadores e restringir superfície em tarefa própria. O parecer também registra token pendente após mudança de papel como tema separado de política de credenciais. |
| AC 1, (g) proteção e relay | Alta, privacidade percebida e i18n. Checkbox em `assets/templates/admin/admin.html:141–145` sem aviso adjacente. O usuário pode entender que proteger a leitura interrompe envios externos. Explicar o efeito e a continuidade do relay mesmo quando ainda não configurado. |
| AC 2 | Alta, acessibilidade funcional. Verificar jornadas completas com JavaScript desativado, links e formulários nativos, foco, teclado e leitor de tela. O código lido confirma controles nativos em parte das jornadas. |
| AC 3 | Média, visual. Capturas responsivas e aceite continuam pendentes. Incluir visitante, comum, dono protegido e root, inclusive erros e estados vazios. |

## 3. Reescopo e dependências propostas

| Issue | Escopo já descrito | Sobreposição e destino sugerido |
| --- | --- | --- |
| 125 | Cartões de identidade, senha e sessão para a conta do administrador. | Ampliar para qualquer sessão segundo doc-041 §7. A página Conta e seus formulários ficam aqui. A 163 define o destino e integra a navegação. A 157 fornece a saída. |
| 126 | Grade de perfis e cartões de perfil/avatar, chaves, importação/exportação, relay e perigo. | Absorver a gestão própria e a lista root de contas/perfis, reaproveitando os serviços da 160/162. Incluir proteção e aviso do relay. A zona de exclusão de conta é root. |
| 127 | Remodelação visual e operacional da configuração do servidor. | Continua root. A 163 retira o link indevido agora e integra o destino existente. Cartões, origem de valores e ações Salvar/Testar pertencem à 127. |
| 128 | Login, revisões e mensagens administrativas. | Ampliar login/resgate e estados 401/403/404 para todos os papéis, preservar as revisões com escopo de dono ou root. A 163 define retorno pós-login e coordena a página de erro com a 157. |
| 157 | Link «Sair…», GET confirmação, POST protegido, cancelamento seguro. | Implementar uma vez para ambos os papéis e revisar a proposta antiga de grupo Administração e retorno fixo ao console. A 163 consome o destino. |
| 153 | Persistir opções restantes do painel de tema em `conf.toml`, configuração global e escrita root. | Manter a persistência na 153. A 163 conserva acesso ao painel de aparência e aplica o controle de permissão de qualquer alteração global. |
| 074 | Tarefa-mãe e padrão responsivo comum. | Atualizar referências históricas à segunda barra segundo a 156. Usar os mesmos cartões, zonas destrutivas e critérios visuais na 125–128. |
| 164 | Validação do corte multiusuário e operação. | Recebe a matriz, executa validação ampla incluindo migração, recuperação, ingestão e relay, e documenta a operação. Depende da 163. |
| 083 | Saúde da ingestão em `/admin/health` por perfil e explicação da origem da escuta. | Preservar seu vínculo à 082. O ponto de navegação futuro deve receber uma permissão explícita. Proposta inicial de console root, com qualquer visualização do dono definida na própria 083 e sem dados de terceiros ou tokens. |

Ordem recomendada após aprovar o estudo

1. Atualizar as tarefas 125–128 e 157 com o contrato e atualizar as referências de navegação da 074. Preservar a 153 e a 127 como trabalhos independentes da liberação da jornada comum.
2. Implementar na 163 o modelo de navegação com as rotas funcionais existentes e a validação do retorno do login. Coordenar com 125 e 157 antes de liberar links para novas rotas.
3. Entregar a confirmação de saída na 157 e Conta na 125. Na integração, ativar o destino pós-login comum e a saída em todas as telas autenticadas.
4. Entregar a 126 e a parte multiusuário da 128. A 126 usa o modelo de navegação e os componentes de página criados na 125. A 128 usa o retorno seguro e a confirmação de saída e pode avançar em paralelo com a 126 após estabilizar essa fronteira. A 163 integra e reúne a evidência dos três ACs.
5. Encaminhar o corte à 164. Evitar tornar a 074 predecessora das filhas que ela agrega. O backlog já mostra um encadeamento histórico 163 → 160/161/162 → 158/159 → 100 → 074; revisar esse vínculo histórico ao atualizar as dependências, especialmente se passar a ligar as filhas à 163.

## 4. Desenho das telas e fluxos

### Login, resgate e erros

- Rota neutra proposta `/login`, mantendo as entradas HTML existentes e seus contratos de formulário. Após autenticar, aceitar apenas destino de leitura conhecido, interno e autorizado para o ator recém-autenticado. Validar o caminho e a permissão além de `SafeReturn`. Se faltar destino ou o acesso for negado, usar `/account` para comum e `/admin` para root. Revalidar toda leitura na rota alvo. Reutilizar a mesma regra para o comum cujo próprio perfil é protegido.
- Resgate usa a tela existente, com rótulos neutros e instrução para solicitar um novo token ao administrador se necessário. Após sucesso, solicitar login normal. Manter respostas sem segredos e a política de referência da página. Emissão root tem instrução para entrega privada ao destinatário.
- Erro 403 definitivo usa o mesmo shell neutro, sem carregar o recurso negado. Para sessão comum, oferecer «Minha conta», «Diretório» e «Sair…». Para o visitante, o 401 HTML orienta a entrar quando aplicável. A falha de leitura protegida usa o mesmo 404 do recurso ausente, com entradas coerentes com a própria sessão. O retorno do 404 não deve distinguir a causa da negativa.
- `GET /admin/sign-out` confirma sem revogar. Exibir «Encerrar esta sessão neste navegador» e botões Sair e Cancelar. Validar o retorno ao cancelar e usar Conta ou diretório como fallback quando adequado. POST `/admin/logout` conserva SameOrigin, CSRF e ticket vinculado à sessão, revoga só a atual e limpa o cookie após sucesso. Renderizar o ticket na confirmação com cache privado sem armazenamento. Sessão expirada volta a um estado de login com explicação neutra. A API de logout mantém seu contrato.

| Estado | pt-BR | en |
| --- | --- | --- |
| Credencial inválida, 401 | Nome de usuário ou senha inválidos. | Invalid username or password. |
| Sessão ausente ou expirada, 401 | Entre para continuar. | Sign in to continue. |
| Ação negada, 403 | Sua conta não tem permissão para acessar esta página. | Your account does not have permission to access this page. |
| Perfil ou página indisponível, 404 | Este perfil ou página está indisponível. | This profile or page is unavailable. |
| Token recusado | Token inválido ou expirado. Solicite um novo token. | Invalid or expired token. Request a new token. |
| Estado de diretório | Protegido | Protected |
| Opção de leitura | Exigir login para ver as escutas | Require login to view listens |
| Ajuda junto à opção | Somente você e administradores podem ver as escutas protegidas. O relay configurado continua enviando escutas ao serviço externo. | Only you and administrators can view protected listens. A configured relay continues sending listens to the external service. |

O log comum de falha deve usar “Login failed” ou “Logout failed”, com motivo seguro quando necessário. As mensagens de API distinguem sessão exigida e ação negada. Evitar dados de credenciais e identificadores secretos no log.

### Conta, navegação e gestões

```text
Diretório
  [Avatar] Nome /slug [Protegido]       [Abrir perfil]
  (+) Visitante: Diretório | Tema e idioma | Entrar
  (+) Comum:     Diretório | Tema e idioma | Minha conta | Meu perfil | Sair…
  (+) Root:      Diretório | Tema e idioma | Minha conta | Meu perfil | Sair…
                 Administração: Contas e perfis | Servidor
Conta
  Identidade [Nome de usuário] [Salvar]
  Senha      [Nova senha] [Confirmar senha] [Alterar senha]
  Sessão    [Sair…]
Meu perfil / perfil selecionado pelo root
  Perfil e avatar [Nome] [Slug] [Avatar]
  [ ] Exigir login para ver as escutas
      Ajuda sobre quem pode ler e continuidade do relay [Salvar]
  Chaves de API [Listar / criar / renomear / revogar]
  Histórico [Importar / exportar]  Relay [Configurar / testar / remover]
Administração — Contas e perfis
  [Criar conta e perfil] [Lista com papel e estado]
  Conta selecionada [Alterar papel] [Emitir token se permitido]
  Perfil selecionado [Cartões de gestão] [Revisar exclusão]
```

- Um ponto no servidor compõe o modelo de navegação para shell público, diretório, gestão e erros. Entradas são identidade da sessão, perfil próprio resolvido pelo servidor, perfil visitado e destino corrente. O modelo consulta `AuthorizeAction` para capacidades da ISSUE-160 e entrega ao template apenas grupos e links autorizados com rótulo, destino e estado atual. Usar a mesma decisão para atalhos de header e ações contextuais. Templates exibem os links recebidos. Servidores de página, ação e serviços continuam validando cada operação e seu alvo.
- Na Conta, o contrato permite trocar a própria senha para comum e root (doc-041 §§2, 5 e 7). Reusar as validações atuais, confirmação e revogação de todas as sessões. Mostrar antecipadamente que será necessário entrar outra vez. Troca do próprio nome de usuário também consta no §7. Manter username separado de slug e nome de exibição. Reset de outra conta ocorre por token root.
- O dono gerencia apenas o perfil próprio, com o alvo derivado da sessão na nova entrada. As rotas legadas com ID continuam verificando o alvo no servidor e preservam ações já permitidas. Importação do dono mantém a exceção do doc-041 §2 para entidades novas. A UI distingue gerenciamento da escuta de edição do catálogo.
- No formulário root para perfil alheio, ajustar a ajuda do checkbox para «Somente o dono deste perfil e administradores podem ver as escutas protegidas». Manter perfis novos públicos por padrão. Garantir que transições de proteção não interrompam ingestão por chave nem relay.
- O header usa o avatar para identificar o perfil visitado. Acesso ao perfil próprio tem rótulo explícito. Indicação de papel root descreve o ator da sessão; evitar sobrepor ao avatar de outra conta. Exibir «Sair…» no menu (+) das duas apresentações de navegação e na Conta, no 403 e na confirmação.

## 5. Acessibilidade e funcionamento sem JavaScript

- Usar links reais, formulários GET/POST do servidor, mensagens renderizadas, confirmação em página e navegação por `details/summary`. Busca e troca de perfil mantêm destinos navegáveis. Verificar alternativa nativa de tema/idioma em navegador sem suporte a popover. O aprimoramento da 124 pode atualizar cartões, preservando todas as operações na resposta completa.
- Foco visível, ordem de Tab conforme a leitura, link para pular ao conteúdo e estado atual com `aria-current`. Ao validar, ligar erros e ajuda aos campos e oferecer resumo com links para os erros. A mensagem e o título precisam ser descobertos na resposta sem JS. Ao usar aprimoramento, preservar ou restaurar foco e anunciar o resultado.
- Ícones decorativos ficam silenciosos. Textos distinguem nome do perfil, estado «Protegido» e ação. Leitor de tela deve perceber estado, relação com a ajuda do checkbox e qual formulário está submetendo. Evitar informação exclusiva por cor ou tooltip.
- Em 320 px, testar sem rolagem horizontal e com zoom e textos longos em en/pt-BR. Garantir contraste de texto, estados, bordas de foco e ícones nos temas Yuu e Pearl, áreas úteis de toque de 44 px e espaço para a barra inferior fixa. Aplicar uma coluna até 40rem e duas quando couber a partir de 64rem conforme a 074.

## 6. Arquivos, esforço e verificação futura

| Parte | Áreas candidatas | Esforço |
| --- | --- | --- |
| Permissões e renderização do shell | `internal/webui/{handlers,live_models,renderer}.go`, modelo de navegação em `internal/webassets/`, `internal/adminui/admin.go`, `assets/templates/shared/nav.html`, `assets/templates/public/{live,media}.html` | G |
| Login, sessão, conta e saída | `internal/adminui/{admin,routing,authorization,actions,messages}.go`, `internal/adminweb/return.go`, registro de rotas em `engine/routes.go`, `engine/handlers/auth.go`, templates admin e traduções en/pt-BR | G |
| Gestão própria e root | `assets/templates/admin/{admin,profile-review,settings}.html`, `internal/adminui/`, `internal/admin/`, tradução e estilos compartilhados em `assets/css/`. A remodelação de Servidor tem sua issue própria | G |
| Cadeado, 404 legado e cobertura | `assets/templates/public/live.html`, `internal/webui/routing.go`, `engine/visibility_issue161_test.go`, tarefa da 161 | M |
| Débitos backend da 162 | `internal/db/{db.go,sqlite/account_tokens.go,sqlite/profile_store.go}` e testes de concorrência/serviço quando encaminhados | M |
| Matriz, capturas e referência | testes existentes de adminui/webui/engine, `scripts/serve_roadmap06.py`, `scripts/roadmap06_fixture.py` e referência visual coberta pelo Makefile | M |

Riscos principais são redirecionar donos para 403 depois de salvar senha, perfil ou importação, quebrar clientes das rotas de gestão legadas, vazar ações root pelo shell ou por HTML compartilhado em cache, e omitir atalhos autorizados de escuta do dono. Testar respostas GET/HEAD, formulários e seus retornos, sem carregar dados de terceiros antes do gate. Preservar destinos válidos usados em revisões e exportação. Cache privado sem armazenamento compartilhado nas páginas personalizadas, com `Vary: Cookie` onde o conteúdo depende da sessão. Inspecionar links, formulários, atributos e dados embutidos na resposta bruta.

| Papel ou estado | Ações e resultados para verificar |
| --- | --- |
| Visitante | Diretório sem derivados, perfil público legível, 404 indistinguível para protegido/ausente, login e resgate navegáveis, ação privada recusada. |
| Comum com perfil público ou protegido | Login válido segue para Conta em ambos os casos, entrada de Conta e Meu perfil, alteração de senha encerra todas as suas sessões, logout encerra só a atual. HTML sem links root, URL global direta recebe 403. |
| Dono em seu perfil | Listagem/escrita própria, chaves, import/export, relay e escutas. Checkbox tem efeito sobre leitura e cache, com ingestão e relay preservados. |
| Comum em perfil de terceiro | Leitura pública possível e leitura protegida indistinguível de ausente. Ausência de links de edição e de dados privados. Chamadas diretas de gestão negadas mesmo com troca de slug, ID ou campos extras. |
| Root | Acesso a todos os perfis protegidos, controle de contas e papel, configuração global, catálogo, imagem legada autorizada, regras de token para terceiros e proteção do último admin. |
| Sessão expirada, revogada ou rebaixada | Retorno seguro, ausência de ações root após rebaixamento, CSRF/ticket inválidos recusados. Duas abas, ticket expirado, reutilizado ou cota saturada. GET de confirmação e prefetch não encerram a sessão. |

Capturar no navegador do Orca com servidor isolado produzido por `scripts/serve_roadmap06.py` (binário local e diretório novo). Conferir as contas fictícias disponíveis e preparar papéis/estados que faltarem na fixture da implementação. Capturar antes/depois em 390, 768 e 1440 px nos temas Yuu e Pearl e incluir 320 px como teste adicional. Percorrer as jornadas sem JavaScript, repetir com aprimoramento ligado e registrar observações manuais de teclado e leitor de tela. Usar `make ui.reference OUT=<diretório novo>` e comparar com a referência vigente antes de atualizar a evidência versionada. Mudanças de template também alteram `data-ui-generation`; revisar diferenças de conteúdo e de layout além do atributo. Aceite visual cabe ao usuário. Testes focados na implementação e validação ampla na 164.

## 7. Decisões abertas e sequência de alterações

1. Aprova `/account` como destino comum, `/admin` para root e retorno prioritário quando a leitura solicitada for autorizada? Recomendo essa combinação para retomar um perfil protegido sem empurrar o dono ao console.
2. Aprova entradas neutras `/login`, `/account` e «Meu perfil» com compatibilidade das rotas atuais? Recomendo introduzir as entradas com a 125/126 e manter os contratos existentes de POST, resgate e exportação.
3. Aprova mover os entregáveis de tela para 125–128 e 157 e concentrar a 163 em integração, permissões, retorno e aceite? Recomendo encaminhar os débitos de lock de resgate e interface de store a tarefas técnicas próprias. Mudança de validade do token ao trocar papel exige decisão separada.
4. Para usuário comum, «Sair…» no menu (+), na Conta e na página 403 é suficiente? Recomendo esse conjunto com confirmação compartilhada; após sair, o diretório oferece «Entrar» e mantém o contexto público.

Sequência de commits ou subtarefas após decisões

1. Backlog e dependências das tarefas envolvidas.
2. Modelo de navegação por ação, compatibilidade de rotas e regressão do HTML servido por papel.
3. Fluxo da 157, Conta da 125 e retorno pós-login da 163. Reviewer verifica status, destinos permitidos, cache, tickets e separação entre sessão e permissão. Se houver alteração de credenciais, bcrypt ou concorrência do resgate, solicitar Advisor (@Sol médio ou @Opus alto) antes de alterar o fluxo.
4. Gestão da 126, mensagens e revisões da 128, aviso do relay e consistência dos 404. Reviewer inspeciona fronteira dono/root e ações diretas.
5. Capturas e referência visual, verificação sem JS, i18n, acessibilidade e aceite visual. Entregar matriz e casos de operação à 164.

Fontes locais consultadas: AGENTS.md e CLAUDE.md, tarefas ISSUE-163/157/125–128/074/153/164 e descrição da 083, doc-041, verificação e pareceres 159–162, consultas ao ai-memory por 163/157/159 e navegação do código com graphify. Os resultados do ai-memory serviram como contexto histórico, com conclusões apoiadas no Backlog, pareceres e fonte atual.
