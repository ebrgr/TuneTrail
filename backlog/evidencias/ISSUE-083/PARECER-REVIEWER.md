# Parecer do Reviewer — ISSUE-083 (commit 58a809e)

Data: 2026-11-21. Diff `main...HEAD`, sem execução de navegador.

Resultado: **APROVADO COM RESSALVAS** (nenhum `[BUG]`; três `[RISCO]` para decisão).

## Verificado
- `go vet ./...` limpo.
- `go test -count=1` de `./engine`, `./internal/adminui`, `./internal/adminweb`, `./internal/db/...` e `./internal/webui`: verdes.
- `make ui.reference.check`: 174 arquivos, sem diferenças.
- Isolamento: todas as consultas filtram por `profile_id`; `TestIngestHealthIsolation` cobre visitante, chave de ingestão, dono (padrão, explícito, outro perfil), root e a página da escuta, e confere ausência de `PRIVATE_*`, URL do relay e chaves.
- Segredos: a tela usa projeções escalares; evidência, snapshot original, token e URL do relay nunca chegam ao template. Texto livre (`client`) é limitado a 120 caracteres e escapado pelo `html/template`.
- Pausa do relay: a regra `paused_until IS NULL OR > agora` é a mesma do painel de relay (`relay_panel.go`), então pausa sem prazo aparece como pausada.
- Identidade visual: nenhum token ou cor nova; reutiliza `.card`, `details`, `graph-values`, `graph-table-scroll` e `admin-sections`. Único CSS novo: `overflow-wrap:anywhere`.

## Achados
- [RISCO] internal/db/sqlite/ingest_health.go:63 (`healthSources`) e :104 (`healthDecisions`) — a janela `row_number()` e o `ORDER BY coalesce(resolved_at,received_at)` percorrem todos os eventos do perfil a cada abertura da página. Em perfis com centenas de milhares de eventos pode ficar lento. Não medi nem conferi o plano de consulta (`EXPLAIN QUERY PLAN`). Sugestão: medir com volume realista antes do aceite final e, se preciso, limitar a janela de tempo ou indexar.
- [RISCO] assets/templates/admin/ingest-health.html:3 — o comentário #2 da tarefa dizia que a página entraria "como item da segunda barra" (ISSUE-123). A implementação põe "Saúde" no seletor `admin-sections` da gestão do perfil, não na barra lateral. Decidir se basta.
- [RISCO] backlog/tasks ISSUE-083, AC #2 — o texto ainda diz "acessível ao admin", mas o acesso implementado é dono do perfil e root (proposta do comentário #3). Ajustar o AC antes do aceite.

## Observações (não bloqueiam)
- Horários aparecem em UTC com sufixo; o restante da interface usa o fuso do perfil em outros lugares. Não verifiquei se há padrão único.
- Pendências já registradas pelo Codex: revisão manual de duplicatas (081b), indicadores da ISSUE-097 e integração com os logs da ISSUE-112.

## Escalonamento
O diff só lê o estado do relay (mensageria), sem alterar entrega nem criptografia. Recomendo ao usuário avaliar o Advisor (`effort` conforme AGENTS.md) apenas se quiser uma segunda leitura de isolamento/segredos.

## Atualização (21/11/2026, após o aceite e a publicação da v0.52)

- Risco 1 (desempenho de `healthSources` e `healthDecisions`): **transferido para a ISSUE-181**, que passa a medir com volume realista e a limitar a janela ou indexar se preciso. Continua sem medição.
- Risco 2 (posição de Saúde no menu): **resolvido**. Saúde entrou no menu (+), acima de Configuração do servidor para root e junto dos links de conta para donos, além do seletor de seções da página.
- Risco 3 (texto do AC #2): **resolvido**. O critério agora diz "dono do perfil e root".
- Observação sobre horários em UTC: mantida, sem decisão.
