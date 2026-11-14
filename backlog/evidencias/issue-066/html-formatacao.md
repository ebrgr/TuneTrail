# ISSUE-066 — Templates legíveis

Solicitação adicional do usuário, antes do aceite: revisar e reformatar o material HTML para facilitar a leitura e edição. Os 12 arquivos em `assets/templates/{public,admin}/` foram reformatados com dois espaços, blocos estruturais e condicionais em linhas próprias, atributos longos desdobrados e chamadas Go longas com argumentos separados. Subexpressões parentetizadas permanecem juntas. Textos e elementos inline sensíveis a espaços permanecem juntos, mesmo quando isso exige uma linha mais longa. O wrapper da retrospectiva, aberto no primeiro item e fechado depois do loop, recebeu comentário explicativo.

Apenas a formatação da ISSUE-067 foi antecipada; unificação de head/sidebar e renomeação de blocos continuam fora deste trabalho.

## Verificação

- Ações Go (exceto o comentário e delimitadores de trim pontuais), sequência de tags, atributos e texto não-whitespace preservados nos 12 arquivos (`html-fontes-verificacao.json`).
- Templates compilados e handlers reais exportados em fixtures fictícias. Referência original da ISSUE-065 intocada; 23 HTML complementares em `html-formatado/`. Os demais 90 arquivos continuam comparados à base original. O gate compara bytes exatos, inclusive whitespace; não foi relaxado para aceitar HTML diferente automaticamente.
- `.gitattributes` preserva o whitespace de fim de linha dos snapshots HTML gerados: ele faz parte da comparação exata. A exceção vale somente para `html-formatado/`, sem afetar as verificações dos arquivos-fonte.
- O comparador recusa overlay com assets ou HTML inexistente na base. Teste verifica também que alterações de CSS e HTML não revisado continuam falhando.
- Chrome local: 23 páginas x seis larguras (320, 360, 390, 768, 1024, 1440) x dois temas, 276 comparações. Texto visível, atributos e posição/tamanho de todos os elementos idênticos. 272 capturas foram idênticas em pixels; quatro apresentaram diferenças de pixels sem diferença de conteúdo/geometria (`html-visual-verificacao.json`, `html-pixels-verificacao.json`). Não se afirma igualdade completa de pixels. Fontes externas bloqueadas igualmente nas duas versões; comparação usa as mesmas fontes disponíveis localmente, sem substituir os arquivos do projeto.
- Verificação complementar das quatro páginas com diferenças, em 320/768 e dois temas: 16/16 capturas idênticas em pixels, com posições de cada caractere visível e de cada elemento idênticas. Dois cenários diferiram apenas em aria-pressed dos botões de tema ocultos, porque a ferramenta aplica o tema depois da carga inicial; detalhe registrado em `html-visual-complementar.json`.
- O primeiro teste visual detectou espaço adicionado em uma mensagem de confirmação; foi corrigido antes da comparação final. Imagens são decodificadas antes da captura.
- `make check` completo sem cache passou: todos os pacotes Go, vet, JavaScript, gate dos 113 arquivos e 12/12 jornadas e2e (`html-check.log`).
- Asserções de links, tabelas, fallback sem JavaScript e valores de configuração agora toleram whitespace de formatação sem perder a verificação dos valores.
- Os alvos Go do Makefile usam `-count=1` porque o cache de resultados não observa confiavelmente as mudanças nos assets externos lidos por `os.Root`.
- Reviewer Claude Opus aprovou o diff final antes do commit (`html-reviewer-final.md`), após corrigir as asserções apontadas no parecer inicial.

## Reproduzir o gate

`make check` e `make ui.reference.check`.

Para inspecionar uma geração: `make ui.reference OUT=/tmp/tunetrail-html-novo`, depois `python3 scripts/compare_ui_reference.py backlog/evidencias/ui-responsiva/base/reference /tmp/tunetrail-html-novo --html-baseline backlog/evidencias/issue-066/html-formatado`.

Não foram alterados CSS, JavaScript, textos de tradução, rotas nem regras funcionais. Sem publicação, merge ou push.
