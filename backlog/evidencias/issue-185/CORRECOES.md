# Correções da ISSUE-185 após a revisão

## Rastro e retenção

A v20 ainda estava restrita à branch da ISSUE-185. Nenhuma tag continha a
primeira versão da migração, e a versão publicada mais recente no checkout era
a v0.59 (schema v19). A correção atualiza a própria v20 e preserva a guarda de
downgrade. Bases de teste já migradas com a primeira v20 precisam ser recriadas
a partir de uma base v19 ou de uma fixture limpa.

O novo ID sequencial permite gravar cada descarte como uma entrada separada.
O teste reenvia o mesmo ID externo confiável com Revisar ligado e descarta o
recibo de revisão duas vezes no mesmo segundo. Os dois registros permanecem
com seus respectivos autores. O ID do registro anterior também distingue uma
recriação do recibo no digest, mesmo quando as versões e o horário coincidem.
Uma confirmação anterior não pode descartar o recibo recriado.

O rastro segue a janela padrão de 90 dias dos detalhes operacionais. A
limpeza oportunística remove até 500 entradas que já excederam a janela do
perfil após um descarte bem-sucedido. Perfis sem novos descartes podem
reter entradas antigas por mais tempo. A exclusão do perfil e a limpeza
global também removem o rastro.

## Descarte do conjunto inteiro

A página mostra a contagem total e pede uma única confirmação explícita
para aquele conjunto. O servidor calcula SHA-256 sobre o perfil e os
recibos elegíveis na ordem de recebimento e ID. Inclui ID, estado,
estacionamento, versões de resolução e trabalho, horário de recebimento
e ID do descarte anterior. O formulário contém só o digest, e a lista
individual continua paginada.

O POST recalcula o digest antes da primeira exclusão. Cada transação aplica
o CAS em até 500 recibos. Entre transações, o escritor pode atender outras
operações. Antes de cada transação seguinte, o servidor compara o restante
da fila com o restante do conjunto confirmado. Mudança ou repetição
retorna conflito. Lotes já confirmados no banco ficam descartados.
Os testes cobrem rollback no segundo lote e expiração de uma lease entre
lotes, que torna um recibo não confirmado elegível e interrompe o descarte.

## Medição

Fixture SQLite com WAL em arquivo temporário, Apple M4, macOS 15.7.3 e Go
1.26.3. São 10 mil recibos estacionados em um perfil, uma confirmação e
20 transações. O tempo de prévia inclui a página de 25 itens e o cálculo do
digest. O tempo do POST inclui a revalidação, os CAS, os gatilhos das projeções
e os commits. Preparação, HTTP e contenção ficam fora da medição.

| Execução | Prévia | POST no repositório |
| --- | ---: | ---: |
| 1 | 11,25 ms | 2,806 s |
| 2 | 11,33 ms | 2,810 s |
| 3 | 11,14 ms | 2,812 s |
| 4 | 10,85 ms | 2,823 s |
| 5 | 10,71 ms | 2,830 s |
| Mediana | 11,14 ms | 2,812 s |

A prévia calcula o digest sem guardar todos os candidatos. O POST mantém
uma cópia dos candidatos em memória e relê os recibos remanescentes em cada
transação para detectar mudanças. Esse custo cresce com o tamanho da fila.
O tamanho maior dos lotes pode aumentar a espera de outros escritores durante
um lote. A medição não quantifica essa espera sob carga concorrente.

```sh
CGO_ENABLED=0 go test ./internal/db/sqlite -run '^TestIngestDiscardScale$' -count=5 -v
```

## Interface

Referência ISSUE-075 regenerada. No Orca, o formulário mostrou a contagem e o
aviso sobre o relay, rejeitou a submissão sem marcar a caixa e ficou válido ao
marcá-la. O CSS tem 73 linhas e usa os breakpoints existentes de 40 e 64 rem.

| Viewport | Largura rolável da página | Largura interna/rolável do cartão |
| --- | ---: | ---: |
| 390 px | 390 px | 356 / 356 px |
| 1100 px | 1100 px | 467 / 467 px |
| 1500 px | 1500 px | 536 / 536 px |

## Verificação automatizada

`TUNETRAIL_WEB_DIR=$PWD/assets make check` passou com 13/13 cenários E2E.
A referência comparou 176 arquivos sem diferenças. Os testes direcionados
com detector de corridas passaram em SQLite e engine, incluindo reenvio
confiável, retenção, concorrência e conflito entre lotes. Grafo AST atualizado.
