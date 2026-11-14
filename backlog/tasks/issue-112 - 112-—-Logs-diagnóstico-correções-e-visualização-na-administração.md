---
id: ISSUE-112
title: '112 — Logs: diagnóstico, correções e visualização na administração'
status: Aberta
assignee: []
created_date: '2026-11-14T03:48:00Z'
updated_date: '2026-11-14T03:54:00Z'
labels:
  - admin
  - operacao
milestone: m-15
dependencies: []
references:
  - backlog/evidencias/logs/diagnostico.md
priority: medium
type: feature
ordinal: 113000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: pedido do usuário em 14/11/2026. Diagnóstico completo em backlog/evidencias/logs/diagnostico.md: o log usa zerolog e vai só para a saída padrão (docker logs), sem arquivo nem retenção; 290 chamadas em debug, 66 em info, 31 em error; cada requisição tem request_id e log de acesso; a senha do reset-password passa pelo logger; logs de debug registram títulos e artistas sem perfil; não há como ver os logs pela interface.

Objetivo: o administrador vê e filtra os logs na administração, sem acesso ao host, sem expor segredos e sem guardar mais do que o necessário. A central de saúde (ISSUE-083) passa a mostrar erros e avisos recentes com link para esta tela.

Etapa 1 — corrigir o que é registrado:
1. reset-password escreve a senha nova direto na saída do comando (fmt), nunca pelo logger.
2. logger.FromContext devolve o logger global configurado quando falta contexto.
3. Remover o RequestIDHook morto.
4. Campos estruturados mínimos em todas as linhas de log: type (access, app, error, audit), component e, quando houver, profile (slug) e request_id; títulos de músicas só em debug.
5. Teste que falha se senhas, tokens, chaves de API, parâmetros Subsonic ou cabeçalhos de autenticação aparecerem em qualquer linha de log das jornadas e2e.

Etapa 2 — captura dentro do servidor:
6. Além da saída padrão, um buffer circular em memória com as últimas N linhas (padrão 5.000, limite de memória fixo), alimentado pelo mesmo logger em formato estruturado, independente do formato do console. Some ao reiniciar: o histórico anterior continua no docker logs.
7. Opcional, desligado por padrão: arquivo em <config>/logs/tunetrail.log com rotação por tamanho (por exemplo 5 arquivos de 10 MB) e retenção documentada; decisão do usuário.

Etapa 3 — tela na administração:
8. Página /admin/logs, só para a sessão de administrador, com no-store: tabela com hora, nível, tipo, componente, perfil, request_id e mensagem; filtros por nível, tipo, perfil, texto e request_id (clicar num request_id mostra a requisição inteira); conteúdo sempre escapado (logs contêm texto enviado por usuários e players).
9. Atualização ao vivo opcional: polling leve agora, SSE quando a ISSUE-084 existir.
10. Baixar o trecho filtrado como JSON Lines.
11. Ajuste temporário do nível (por exemplo debug por 30 minutos, voltando sozinho), com CSRF e registro de auditoria; o nível persistente continua na página de configurações.
12. Entrada no submenu de administração (ISSUE-110) e, na central de saúde (ISSUE-083), contagem de erros e avisos da última hora com link para a tela filtrada.

Fora do escopo: envio para serviços externos (Loki, syslog) e métricas (ISSUE-104).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Senha do reset-password não passa pelo logger; teste e2e confirma que nenhum segredo aparece nos logs
- [ ] #2 Todas as linhas com type e component; profile e request_id quando aplicável; logger sem contexto respeita nível e formato
- [ ] #3 Buffer em memória com limite de linhas e memória, independente do formato do console
- [ ] #4 /admin/logs só para administrador, com filtros por nível, tipo, perfil, texto e request_id, conteúdo escapado e no-store (testes de permissão e de escape)
- [ ] #5 Download do trecho filtrado em JSON Lines
- [ ] #6 Nível temporário com volta automática, CSRF e auditoria
- [ ] #7 Decisão do usuário sobre o arquivo em disco registrada; se ativado, rotação e retenção testadas
- [ ] #8 Central de saúde mostra erros e avisos recentes com link (quando a ISSUE-083 existir)
- [ ] #9 Linhas registradas durante uma requisição, inclusive no trabalho em segundo plano iniciado por ela, saem com o request_id (teste); RequestIDHook removido
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-14T03:54:00Z

---

Esclarecimento sobre request_id e RequestIDHook (14/11/2026, a pedido do usuário).

request_id: código aleatório de 8 caracteres gerado por requisição (WithRequestID, engine/middleware/middleware.go), gravado nas linhas de log daquela requisição e devolvido no cabeçalho X-Request-ID. Serve para ligar todas as linhas de um mesmo acontecimento (ex.: uma escuta com erro 500 mostra, filtrando pelo código, a sequência associação de álbum → consulta MusicBrainz → falha ao salvar). Na tela /admin/logs, clicar num request_id filtra a requisição inteira. Hoje chega ao log pelo logger da requisição que o middleware Logger injeta no contexto (baseLogger.With().Str("request_id", ...)); quem usa logger.FromContext(ctx) já o grava.

RequestIDHook (middleware.go:20): hook do zerolog pensado para acrescentar o request_id a qualquer linha registrada com contexto (.Ctx(ctx)), inclusive pelo logger global — cobriria trabalho em segundo plano (ex.: fila de imagens) e código que usa logger.Get() com ctx em mãos. Não faz nada hoje: (1) nunca é registrado (nenhum .Hook(RequestIDHook{})); (2) procura ctx.Value("requestID") com string simples, enquanto o middleware guarda com a chave tipada MiddlwareContextKey("requestID"); em Go as chaves comparam também o tipo, então nunca coincidem. Provável herança incompleta do Koito.

Decisão recomendada para a Etapa 1: remover o RequestIDHook e propagar o contexto com o logger da requisição para o trabalho em segundo plano. Acrescentado critério: linhas registradas durante uma requisição, inclusive em segundo plano, saem com o request_id (teste). Alternativa descartada salvo necessidade: consertar o hook (registrar no logger global e usar a chave tipada).

---
<!-- COMMENTS:END -->
