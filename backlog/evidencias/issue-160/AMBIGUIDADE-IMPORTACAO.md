# ISSUE-160 — decisão pendente sobre metadados de importação

Análise estática em 18/11/2026. Implementação de autorização interrompida conforme a instrução explícita do executor: registrar e parar se a matriz for ambígua ou conflitar com o código. Nenhum código de runtime foi alterado.

O doc-041 §2 permite ao usuário comum importar histórico do próprio perfil, mas reserva “capas, aliases, merges e outras alterações do catálogo compartilhado” ao root. O §4 mantém a ingestão por chave de máquina. O contrato não explicita a fronteira entre criação automática de identidade musical durante ingestão/importação e aplicação de metadados editoriais fornecidos num arquivo de exportação.

## Evidência

`internal/importer/koito.go`, `ImportKoitoFile`:

- Linhas 23–27: exige profile_id, guarda o destino das escutas e remove o escopo pessoal para resolver catálogo compartilhado.
- Linhas 95–111: quando o artista não existe, cria-o com ImageSrc e todos os aliases fornecidos pelo arquivo.
- Linhas 133–149: quando o álbum não existe, cria-o com ImageSrc, aliases e VariousArtists fornecidos pelo arquivo.
- Linhas 171–184: quando a faixa não existe, cria-a e grava todos os aliases fornecidos pelo arquivo.
- Linhas 192–199: a escuta recebe o profileID selecionado. Isolar somente esse destino não impede a criação dos metadados compartilhados acima.

Esses trechos criam entidades novas; esta análise não afirma que substituam capas ou aliases de entidades já existentes. A dúvida é justamente se essas criações constituem uma exceção permitida à exclusividade root.

O grafo foi consultado antes da leitura de fontes, com as consultas sobre autorização/rotas e ImportKoitoFile/SaveTrackAliases no graph.json externo indicado pelo CLAUDE.md.

## Decisão necessária

Escolher a regra para importação feita pelo dono:

1. Permitir criação automática de entidades musicais, mas ignorar capas e aliases adicionais do arquivo; manter apenas a identidade mínima necessária para registrar a escuta. Root mantém a importação completa. Essa opção reduz a fidelidade do restauro feito por usuário comum.
2. Permitir também capas e aliases na criação de entidades novas pela importação, como exceção explícita à regra root; alterações editoriais de entidades existentes seguem exclusivas do root.

Não foi escolhida uma regra implicitamente. Os endpoints permanecem sob a exigência administrativa existente. Não foram executados make check ou testes de corrida, pois não houve implementação. Nenhum AC foi marcado. ISSUE-160 permanece Em andamento; sem push, merge, publicação ou Done.

## Decisão do usuário

Opção 2 autorizada explicitamente: a importação do dono pode criar entidades novas com capas e aliases. Alterações editoriais de entidades existentes continuam exclusivas do root. Implementação retomada; o registro de interrupção acima descreve somente o momento anterior à decisão.
