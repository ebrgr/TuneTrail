# 🛡️ Parecer do Advisor: REPROVADO COM RESSALVAS (plano viável; três pontos precisam ser resolvidos antes de implementar)

**Limites desta revisão:** foi somente leitura e nenhum arquivo foi alterado. Não li `estudos-locais/`. Só tive Read, Grep e Glob, então não consegui rodar `graphify query` como o AGENTS.md pede. Localizei o código com buscas pontuais; quem implementar deve rodar o graphify antes de explorar mais.

## Falhas críticas do plano como está escrito

1. **"Byte a byte" não é possível hoje sem mascarar valores voláteis, e há quatro fontes deles:**
   - **CSRF:** `adminweb.NewCSRF` gera uma chave aleatória (`internal/adminweb/csrf.go:32`), e o nonce de login também é aleatório (`:122`).
   - **Tickets de submissão, planos de catálogo e avisos:** são UUIDs e usam `time.Now()` (`internal/adminui/submission.go:22`, `admin.go:500`, `settings.go:104`). A administração não tem relógio injetável.
   - **Prévia:** `makePage` usa `time.Now()` direto (`internal/webui/preview.go:142`), então a grade de atividade muda todo dia.
   - **Versão:** `version.Current` aparece no HTML e muda quando a 066 for publicada.

   As páginas públicas já aceitam `Options.Now` (`internal/webui/handlers.go:34`), então essas podem ficar determinísticas.

2. **`capture_admin_webui.cjs` não serve como está para gerar a base:**
   - ainda tem o modo `react` e a data fixa `'2026-11-05'`;
   - usa o slug `mara`, mas o `roadmap06_fixture.py` cria `alpha`, `beta` e `empty`. `mara` só existe depois que `validate_admin_webui.py` renomeia o perfil;
   - **altera os dados**: remove e adiciona alias e troca a capa;
   - cobre só a administração, em duas larguras;
   - libera `fonts.googleapis.com`, o que deixa a captura dependente da rede.

3. **As capturas tiradas do binário não têm relógio controlável.** As páginas com período padrão ("semana") e os tempos relativos variam conforme o dia em que se roda.

## Arquitetura proposta

### A. Referência HTML/CSS: teste Go que fica desligado por padrão

- **Onde:** um teste novo e isolado (por exemplo `engine/uireference_test.go`, ou um pacote `internal/uireference` só com teste). Ele chama os construtores públicos (`webui.NewPreview`, `webui.NewHandler`, `webui.Assets`, `webui.PublicFiles`, `adminui.New`) **por HTTP com `httptest`**, nunca lendo os templates diretamente. Assim ele continua funcionando quando a 066 mover tudo para `assets/`.
- **Quando roda:** só com `TUNETRAIL_UI_REFERENCE_OUT=<dir>` definido; sem isso, o teste é pulado (`t.Skip`). Não deve virar uma comparação permanente no CI, porque a 069 e a 070 mudam a interface de propósito.
- **Dados:**
  - **Recomendado:** reaproveitar o `roadmap06_fixture.py`. O teste recebe `TUNETRAIL_UI_FIXTURE_DB` apontando para o `go/koito.db` gerado e trabalha numa cópia em `t.TempDir()`. Assim o HTML de referência e as capturas usam exatamente os mesmos dados.
  - **Alternativa:** semear um `sqlite.NewInMemory()` em Go, mas isso duplica a semântica da fixture.
- **Como tornar determinístico sem mexer no runtime:**
  - `Options.Now` fixo em `2025-06-01T12:00:00Z` (o `REFERENCE_NOW` da fixture, que hoje está marcado como "não injetado");
  - `TUNETRAIL_FORCE_TZ=America/Sao_Paulo` e `ForcedTimezone` ativado;
  - catálogo de idiomas e `Accept-Language` fixos;
  - `version.Current = "0.0.0-reference"`, atribuído só no teste (é uma `var`);
  - sessão de administração criada via store, como já faz `catalog_review_test.go:27`.
- **Mascaramento na hora de gravar**, com regras ancoradas no nome do atributo:
  - os valores de `csrf_token`, `submission_token` e `plan` viram `«CSRF»`, `«TICKET»` e `«UUID»`;
  - na prévia, só os atributos `title` e `opacity` das células de atividade;
  - `Set-Cookie` não é gravado.

  O manifesto registra **quantas substituições cada regra fez**. Se o número sair do esperado, o teste falha, para que a máscara não esconda uma regressão.
- **Cobertura mínima, com uma tabela rota → template:** o teste lista os `{{define}}` de primeiro nível e **falha se algum template não estiver mapeado**.
  - **Prévia:** `/`, `/demo`, `/luna`, `/iara`, os estados `empty`, `error`, `long` e `broken`, `theme=pearl` e `weekstart=mon`.
  - **Público:** seleção de perfil; `/alpha?period=all_time`; os três rankings; histórico páginas 1 e 2; artista, álbum e faixa 1; faixa 4 (nome longo em Unicode); álbum 2 (imagem ausente); rewind `year=2024&month=5` e o anual; busca; perfil `empty`; 404 e 400 de filtro inválido.
  - **Administração:** login, perfis, conta, configurações, chaves (valores ocultos), importação e exportação, relay, revisão de exclusão de perfil, edição de artista, álbum e faixa, revisão de mesclagem e exclusão, confirmação de escuta.
  - **Arquivos estáticos:** `ui.css`, `themes.css`, CSS da administração, todos os JS, `themes.json` e o favicon, gravados com o `Content-Type` servido.
- **Manifesto:** `manifest.json` com commit, `go version`, `fixture-report.json`, as regras de máscara e suas contagens, e o SHA-256 de cada arquivo. Vale conferir o hash do `ui.css` contra `65e367f9…`, registrado em `fontes-retrospectiva.md`.

### B. Comparação: `scripts/compare_ui_reference.py <base> <nova>`

- Usar só a biblioteca padrão e rodar com `python3 -I`. A saída é um relatório JSON e o código de saída indica o nível:
  - **0:** idênticos byte a byte, depois de reaplicar a máscara (a máscara é idempotente);
  - **1:** idênticos só depois de normalizar;
  - **2:** diferentes, ou arquivo faltando ou sobrando.
- **Normalização:** troca qualquer sequência de espaços por um só e remove o espaço entre `>` e `<`.
  - **Ressalva:** isso pode esconder uma mudança real entre elementos inline (o doc-031 já alerta). Por isso, "normalizado igual" é necessário, mas não basta para aceitar a 067; as capturas continuam obrigatórias.
- **CSS:** na 068, comparar o CSS concatenado como é servido, com uma normalização própria (remover comentários e espaços). Mudança de ordem conta como diferença.

### C. Capturas: script novo `scripts/capture_ui_base.cjs`

Recomendo um script novo, que reaproveita as funções `goto`, `shot` e a verificação de overflow do atual. Mexer no script atual quebraria a reprodutibilidade das evidências antigas.

- **Matriz:** larguras {320, 360, 390, 768, 1024, 1440} × temas {yuu, pearl}.
  - `deviceScaleFactor: 1`, altura fixa, `reducedMotion`, `animations: 'disabled'`, `timezoneId` e `locale` fixos;
  - `page.clock.setFixedTime(REFERENCE_NOW)` para o JS do navegador;
  - **sem nenhuma alteração de dados** e com uma fixture nova a cada execução;
  - só rotas com período explícito (`all_time`, `year=2024`), já que o servidor usa o relógio real;
  - servidor iniciado pelo `serve_roadmap06.py`, que já recusa diretório existente e sobe só em `127.0.0.1`.
- **Volume:** seriam cerca de 25 páginas × 12 combinações. Sugiro capturar só o viewport (como o script atual), não a página inteira, e restringir as capturas a cerca de 10 páginas representativas. O HTML cobre todas as páginas. Antes de gravar, combinar com o usuário o tamanho que vai para o Git (Git LFS ou não).
- **Fontes:** registrar uma de duas formas:
  - (a) rodada com Google Fonts liberado, que é o comportamento atual e fica como base;
  - (b) se o usuário aprovar, uma rodada com fontes externas bloqueadas, que mostra o fallback e serve de "antes" para a 070.

  Em cada uma, guardar `document.fonts.check(...)` e se as fontes chegaram a carregar.
- **Referência tipográfica da retrospectiva:** temas yuu e pearl, nas larguras **390, 768 e 1440**. Para o título, o vencedor (artista e álbum), os rótulos de seção, o ano, os números e os controles, gravar em JSON a família, o peso, o tamanho, a caixa (`text-transform`) e o `line-height` computados. Gravar também **a fonte realmente usada**, via CDP `CSS.getPlatformFontsForNode`. É isso que fecha a lacuna que a 061 deixou ("não houve medição da fonte efetivamente usada"). Junto, um PNG recortado do bloco.
- **Destino:** `backlog/evidencias/ui-responsiva/base/{html,css,capturas,tipografia}/` e um `screenshots.json` com a matriz, o Chrome, o hash da fixture, a rota e o status de overflow.

## Riscos e casos de borda

- **`mara` versus `alpha`:** é preciso decidir um conjunto de dados para a base inteira. Recomendo `alpha` sem renomear.
- **Configurações da administração:** `Options.Settings` nulo pode quebrar a página `settings`. Validar ou passar um store de configurações temporário.
- **Tempos relativos na administração** (chaves, sessões) dependem do relógio real, então precisam de máscara ou de dados sem idade relativa.
- **Fixture:** o hash do SQLite muda com a versão do SQLite. A comparação deve usar as contagens do relatório, não o `base_sha256`.
- **Pixels entre máquinas:** variam por versão do Chrome e por renderização de fontes. As capturas servem de referência visual e para aceite, não como teste automático de pixels.
- **A 066 é mudança de runtime** (doc-031): o comparador precisa rodar contra a referência gravada **antes** da mudança, não contra uma referência gerada de novo.
- **Segurança:** a fixture tem uma senha fictícia pública. Os servidores devem subir só em loopback, e nenhuma captura pode mostrar valor de chave ou token de relay.

## Passo a passo (tudo dentro da ISSUE-065, tipo chore, sem mudança de runtime)

1. Rodar `graphify query` sobre os handlers de webui e adminui e as rotas em `engine/routes.go`, para confirmar a lista de rotas e templates.
2. Escrever o teste de referência (seção A) com a tabela rota → template e as regras de máscara.
3. Escrever `compare_ui_reference.py` com um teste próprio (`scripts/test_compare_ui_reference.py`) para os casos idêntico, normalizado, diferente e faltando.
4. Adicionar um alvo `make ui.reference` (fixture → teste Go → manifesto) e um `make ui.compare BASE=… NEW=…`.
5. Rodar duas vezes seguidas e confirmar **código 0**. Esse é o critério de reprodutibilidade.
6. Escrever `capture_ui_base.cjs` com a matriz, a tipografia e o JSON (seção C).
7. Gravar a base, ligar o diretório à tarefa com `--add-ref`, marcar os critérios de aceite #1 a #3 só depois de verificar e mudar o status para "Aguardando aceite" com um comentário de verificação.
8. Entrada no `CHANGELOG`: dispensável, porque não há mudança visível ao usuário.

## Decisões que ficam com o usuário

1. **Prévia:** mascarar a grade de atividade (o que recomendo, pois mantém o runtime intacto) ou criar um ponto de injeção de relógio, o que seria uma mudança fora do escopo da 065.
2. **Rodada com fontes externas bloqueadas:** incluir ou não.
3. **Volume das capturas:** Git LFS ou não, e quantas páginas capturar.
