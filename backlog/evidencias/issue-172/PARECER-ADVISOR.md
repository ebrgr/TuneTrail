# Parecer do Advisor — ISSUE-172 fase 2 (effort high)

**APROVADO COM MELHORIAS** (nenhuma bloqueia o corte; uma exige issue de acompanhamento).

Escopo analisado: migração 021, remoção do interruptor/escritor legado, resolvedor por ocorrência, relay em rascunho, importadores de arquivo, medições e prova de restauração. Premissa do usuário: produção sai da v2 direto para a última versão.

## Falhas críticas
Nenhuma.

## Análise por eixo
1. **Migração (concorrência e dados):** o salto v2 → v21 foi provado com 200 escutas e digest idêntico, e a cópia real de 2.826 escutas restaurou com contagens e digests iguais. A 021 roda numa transação do goose; o custo é proporcional às linhas de `ingest_events` abertas (o UPDATE de `relay_admission_generation` filtra `closed_at IS NULL`), então não há reescrita de tabela grande. A guarda recusa recibos de sombra sem vínculo, que só existem em bancos de desenvolvimento. Nenhum risco para produção.
2. **Segurança do relay:** o rascunho elimina a saída de rede antes da aprovação, e escutas anteriores à aprovação nunca são enviadas depois (a geração de admissão do recibo impede). A confirmação reresolve DNS e recusa uma resposta que o aprovador não viu. Relays já configurados na atualização continuam ativos (`awaiting_approval` padrão 0); mudar URL ou token volta ao rascunho. Coerente.
3. **Identidade:** a regra do MBID de artista (um dono recebe a escuta, 2+ vira `pending_identity`) preserva a paridade com a ISSUE-170. MBIDs declarados por arquivo entram sem sobrescrever MBID diferente; a proveniência `legacy_declared` do recording segue a ISSUE-187 item 4 (validação posterior pelo MusicBrainz).
4. **Escalabilidade:** os gatilhos novos em `ingest_events` (inbox, resumo por fonte, geração de admissão do relay) custaram pouco: ACK p95 11,4–11,6 ms a 15/s contra 3,6 ms antes, com visibilidade máxima de 1,07 s. Folga grande sobre os limiares (200 ms, 10 s).
5. **Regras do projeto:** issue-first, plano, decisões registradas, CHANGELOG em inglês e comentários em inglês cumpridos.

## Melhorias
1. **Remover o código de compatibilidade com schema < 21** (resolvedor v9, ativação, ramos de `SaveListen`, consultas a `profile_ingest_flags`, `catalog.SubmitListen`) e reescrever os testes de atualização para semear o schema 20 por SQL em vez do escritor antigo. É o que falta para cumprir ao pé da letra o critério 1. Não afeta produção (v2 → última); recomendo uma issue de acompanhamento, sem bloquear o aceite.
2. **Acompanhar Revisar ligado por padrão** nos primeiros dias (volume de `pending_duplicate` por fonte) e ter à mão o botão de aceitar em lote da ISSUE-181.
3. **Runbook do corte:** executar exatamente `backup .backup` → upgrade → `quick_check` → conferir contagens/digests, janela de 10 minutos, como em `FASE2.md` e na documentação de atualização; não há reversão sem o backup.
4. **Decisão de produto ainda aberta:** reenvio de ID externo confiável com Revisar ligado (hoje abre pendência).

Decisões tomadas pelo Claude com padrão conservador durante a fase 2 (reversíveis pelo usuário): escuta manual informa "já existe"; aliases secundários do Koito v1 preservados com orçamento; truncagem de opcionais e recusa individual acima de 10 KiB; MBIDs declarados vinculados offline com proveniência declarada; MBID de artista honrado; relay como rascunho até a aprovação.
