# ISSUE-061 — Parecer Claude Code CLI / Opus

Data: 14/11/2026. Invocação real concluída com código 0, Opus, effort medium, ferramentas Read/Grep/Glob, somente revisão documental. Teste prévio respondeu ADVISOR_OK. A sandbox informava ausência de login; fora dela a sessão existente foi reconhecida. Não foram copiadas credenciais.

O texto abaixo é o parecer do Advisor. Os apontamentos são propostas de revisão, não mudanças aplicadas ou decisões novas do usuário.

### 🛡️ Parecer do Advisor: REPROVADO COM RESSALVAS

O desenho está correto e segue as decisões do usuário: tudo em `assets/`, HTML em `assets/templates/`, leitura do disco em produção e desenvolvimento, fontes locais e embed só como opção futura. O snapshot por geração, o root sem fallback silencioso e a publicação apenas de subpastas autorizadas fecham os riscos principais. A reprovação vem de três pontos de documentação e governança, corrigíveis sem refazer o estudo: dados da instalação real gravados na evidência, um alvo visual que o usuário não decidiu e ACs da 065 que não cobrem o que a 070 exige.

Escopo da revisão: li só os documentos pedidos (AGENTS.md, advisor.md, doc-031, a evidência e as tarefas 061, 065, 066 e 070). Não li código, estudos-locais nem dados reais. Não alterei nada e não fiz consultas externas.

---

#### Falhas críticas (problemas no desenho ou no registro, a corrigir antes do aceite)

1. **Dados da instalação real na evidência versionada.** `fontes-retrospectiva.md:8` traz um IP de rede local e o caminho de um perfil real (`/<usuário>/rewind`). O arquivo fica em `backlog/evidencias/`, que é versionado.
   - **Correção mínima:** trocar por `<instância local TuneTrail 0.10>/<perfil>/rewind` antes de qualquer commit. Se já houver commit, avisar o usuário.

2. **O alvo visual virou decisão sem aceite do usuário.** A AC #7 da 061 pede só *investigar* as diferenças. A 070, porém, manda "corrigir" o TuneTrail para os valores da referência Koito (vencedor 600 sem uppercase, título 36 px, rótulos 16/400). Nada registra que o usuário escolheu igualar a referência.
   - Isso pesa mais porque o título tem tamanho máximo de 42 px (`:1153`, provavelmente `clamp`). Foi medido numa única largura (~1487 px), com tema, idioma e dados diferentes. Os 42 px contra 36 px podem ser intencionais ou depender da largura.
   - **Correção mínima:** na 070, escrever "alvo tipográfico sujeito à confirmação do usuário após a referência controlada da 065". Medir o título em pelo menos 3 larguras antes de fixar valores.

3. **A 065 não garante o que a 070 exige.**
   - A tipografia da retrospectiva (família, peso, tamanho, caixa, line-height e fonte renderizada) aparece só no Implementation Plan da 065. Não está nos Passos nem nos ACs.
   - A descrição cita "doc-031 §Como verificar, passo 1", seção que não existe mais no doc-031 reescrito.
   - **Correção mínima:** acrescentar o AC "#4 Referência tipográfica da retrospectiva registrada no mesmo tema, viewport e dados fictícios, em pelo menos 3 larguras". Trocar a origem para "doc-031 §Sequência de execução e validação".

#### Problemas de desenho menores (corrigir no texto)

4. **Fontes sem hash na URL quebram a invalidação de cache.** O CSS concatenado recebe hash, mas os `url()` do `@font-face` em `fonts.css` vão apontar para `/web-assets/fonts/x.woff2` sem hash. Se a fonte mudar e for servida com cache longo, o navegador continua com a antiga; com `no-cache`, perde-se o ganho.
   - Na 068/070, os `url()` do CSS também precisam receber o hash. Registrar como requisito da 106.

5. **Licenças dentro de pastas servidas contradizem a regra de exposição.** O doc-031:54 diz que licenças internas não ganham rota. A árvore proposta, porém, põe licenças em `icons/` e `fonts/`, que são servidas.
   - Definir o manifesto por extensão (`.woff2`, `.svg`, `.css`, `.js`, `.json`), não por diretório. Os mesmos textos de licença devem ir para a imagem final e o pacote, como a OFL exige.

6. **Nome Reservado de Fonte (RFN) da OFL.** Converter TTF para WOFF2 ou criar subconjunto conta como "Modified Version" na OFL. Se o copyright declarar RFN, a versão modificada não pode usar o nome original.
   - Na 070, preferir WOFF2 já publicado pelo upstream. Se for converter, verificar RFN no texto da licença de cada família, incluindo Jost, League Spartan e Inter, e registrar isso no inventário.

7. **Recarga de templates do disco abre um vetor de injeção.** Em produção, quem puder escrever no root controla templates que executam métodos sobre os dados.
   - O "separar de diretórios graváveis" do doc é vago. Na 066, exigir root com dono root e só leitura para o usuário não-root do container (coordenar com a ISSUE-102), com teste ou checagem no Dockerfile.

8. **Ambiguidade sobre symlinks.** Os symlinks da 066 tratam do executável. Não há regra para symlinks *dentro* de `assets/` que apontem para fora do root.
   - Especificar: o snapshot lê só pelo manifesto, com acesso confinado ao root (por exemplo `os.Root`/`fs.Sub`), e recusa links que escapem. A busca nas requisições é por chave de mapa do snapshot, nunca por caminho do disco.

9. **Validar só o parse não basta.** O parse do `html/template` não pega `{{template "x"}}` inexistente nem erro de execução. Partir do servidor não prova que as páginas renderizam.
   - Acrescentar o smoke test de renderização da 065 como gate de CI da 066. Manter conjuntos de templates separados para público e administração, para evitar colisão de nomes curtos entre `public/` e `admin/`.

10. **Catálogos `langs/` fora do root único.** A exclusão é coerente (não são arquivos web), mas a 066 não diz como `langs/` é localizado. Exigir as mesmas regras do root de assets (executável real, sem cwd, explícito inválido falha), ou declarar o contrato próprio por referência.

11. **Inconsistências menores no registro.**
    - O arquivo da 066 ainda se chama `…Mover-templates-e-assets-para-templates-e-assets-na-raiz.md`, embora o título tenha mudado.
    - A 070 omite o "Ano" da tabela de evidência (referência Jost 20/500, TuneTrail League Spartan 20/600).
    - A AC #3 da 061 (comparar localizações) está marcada, mas a comparação saiu do doc reescrito. Basta uma linha apontando o histórico do Git.

#### O que está correto

- A conclusão da comparação está bem fundamentada: peso e caixa vêm da herança de `h3`, não de falha de download. Trocar só a origem das fontes não iguala as páginas.
- As limitações estão declaradas: não houve medição da fonte efetivamente renderizada, e a referência estava em tema claro contra escuro na instalação.
- O inventário SFNT é correto: TTF estáticos, sem declarar faixa variável.
- A 066 classificada como mudança funcional com versão própria.
- Nenhum embed antecipado e nenhum fallback silencioso.

#### Validação pendente, que depende de ler código (não verificada nesta revisão)

- Se o Dockerfile atual copia `assets/` só no estágio de build ou também na imagem final. O doc-031:22 e o :78 sugerem respostas diferentes.
- Se existe CSP permitindo `fonts.googleapis.com` ou `fonts.gstatic.com`. A 070 deve endurecer para `font-src 'self'`.
- Os tipos MIME servidos para `woff2` e se há `nosniff`.
- O comportamento atual de cache e hash em `/web-assets/...`, para a 066 não regredir antes da 106.
- Todos os consumidores atuais dos TTF, além de `internal/summary/image.go`.
- A cadeia de dependências 066 → 067 → 068 → 070. Não li a 067 nem a 068.

#### Plano de correção mínimo

1. Anonimizar a URL da instalação real na evidência.
2. Na 070: marcar o alvo tipográfico como sujeito ao aceite do usuário, medir em várias larguras, incluir o "Ano" e verificar RFN/licença por família.
3. Na 065: acrescentar o AC de referência tipográfica e corrigir a seção de origem.
4. No doc-031 e na 066:
   - manifesto por extensão, com licenças fora de rota;
   - root só leitura com dono root;
   - confinamento de symlinks ao root;
   - smoke test de renderização como gate;
   - contrato de `langs/`;
   - `url()` de fontes com hash, coordenado com a 106.
5. Registrar este parecer como comentário na ISSUE-061, sem mudar status. O aceite continua com o usuário.
