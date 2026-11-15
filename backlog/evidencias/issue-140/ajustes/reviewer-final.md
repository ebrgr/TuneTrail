Não consegui rodar `go test ./engine/ -run '140|139'`: o comando ficou aguardando sua aprovação. Esta revisão é só leitura do diff e do código ao redor. Rodar esse teste é o primeiro passo antes do commit.

## Revisão do diff da ISSUE-140

**Parecer: aprovado.** Não encontrei bug nem falha de segurança que impeça o commit. Abaixo, o que conferi e alguns ajustes pequenos que recomendo.

### O que conferi e está correto
- **Ticket da busca dinâmica** (`internal/adminui/media.go:450-455`): o servidor só reaproveita o valor de `X-TuneTrail-Submission` se ele for um UUID válido. Ele não cria tickets novos durante a busca, e o ticket é validado de novo no envio. Na mesclagem não há ticket nenhum. Não tem como abusar disso.
- **`data-section` dentro de outra seção**: dentro do diálogo, `request()` troca o conteúdo inteiro pela seção `#task` (`assets/js/media-dialog.js:91-98`). Por isso o `#artist-results` com `data-section`, colocado depois de uma busca, não confunde o envio seguinte. Na página administrativa completa, ele só ganha `data-section` quando é a própria resposta da busca, e o script de validação testa esse caso.
- **Respostas fora de ordem**: `searchVersion`, o `AbortController` e a checagem `owner === activation` descartam respostas antigas. `cancelSearch()` é chamado em `clear()` e em `request()`, então nenhuma busca sobrevive depois que a tarefa fecha ou muda.
- **Fuso opcional**: o formulário antigo, com `timezone` preenchido, continua funcionando. Um `tz` inválido em `return_to` é rejeitado antes de cair no padrão silencioso de `webquery.Timezone`.
- **Traduções**: `admin.text.existing_artist_id` e `admin.text.utc_offset_for_ambiguous_time` já existem nos dois idiomas, e as chaves novas foram adicionadas em ambos.

### Ajustes recomendados (nenhum impede o commit)
1. **Botões só com ícone de lixeira** (`manage.html`, aliases e artistas): quem usa mouse não vê nenhuma dica do que o botão faz. Além disso, o `aria-label` "Remove alias" se repete igual em todas as linhas, enquanto o botão de artista já inclui o nome. Sugestão: usar `trn` com o nome do alias e adicionar um `title`.
2. **`<details class="listen-offset" {{if .Error}}open{{end}}>`**: o campo de deslocamento abre com qualquer erro, inclusive o de fuso inválido, e aí distrai do problema real. Seria melhor abrir só no conflito de horário ambíguo (`db.ErrConflict`), por exemplo com um campo próprio no `mediaView`.
3. **Busca de mesclagem vazia**: quando `merge_query=""`, aparece "No matching items. Try another name.". Com o campo vazio, o certo seria não mostrar nada ou mostrar uma dica.
4. **Tooltip em telas de toque** (`tooltip.css`): no celular, o `:hover` continua ativo depois do toque, então o tooltip pode ficar visível ao fechar o diálogo. Sugestão: envolver a regra de `opacity:1` da `.media-actions` em `@media (hover:hover)`, mantendo o `:focus-visible` fora dela.
5. **`time.LoadLocation` aceita `"Local"`**: com esse valor explícito, a escuta seria registrada no fuso do servidor. É um caso raro, mas dá para rejeitar `Local` junto com o erro de fuso inválido.
6. **Detalhes menores**:
   - `len(v.Name) <= 100` conta bytes, enquanto o `maxlength` do campo conta caracteres. Nomes com acentos perto do limite deixam de vir preenchidos, sem outro efeito.
   - Em `public/media.html`, as linhas novas de `data-tip`/`title` estão com indentação diferente das de `aria-label`.

### Antes do commit
- Rodar `go test ./engine/ -run '140|139' -count=1` e o `validate_media_dialog_refinements.cjs` com uma fixture nova, guardando as evidências em `backlog/evidencias/issue-140/`.
- Depois, rodar `analisar-tunetrail.sh` para atualizar o grafo.

Quer que eu aplique algum desses ajustes ou que rode os testes, se você aprovar o comando?
