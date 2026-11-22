# Plano ISSUE-186 — aceitar escuta sem álbum como Lançamento desconhecido

Decisão do usuário (22/11/2026) após o ensaio da fase 1 da ISSUE-172. Evidência: `backlog/evidencias/issue-172/ENSAIO.md` (205 de 2.250 requisições sem álbum viraram `pending_identity / release_missing`; o legado as gravou).

## Contrato (doc-039, decision-008)
"Álbum ausente: não inventar álbum usando o título da faixa. Ocorrência provisória sem lançamento (release_id nulo, origem unknown), ligada à gravação quando identificável. Nenhuma atribuição automática a um álbum conhecido da gravação. A escuta conta no total e na gravação, mas em nenhum álbum real; balde 'sem lançamento' na reconciliação. Se o álbum surgir depois, revisão explícita pode reassociar o evento e preservar o snapshot anterior. Fonte com ID estável de item pode reutilizar a ocorrência provisória; sem contexto suficiente, não se presume que duas ocorrências provisórias são a mesma."

## Objetivo
A ingestão por ocorrência aceita a escuta sem álbum (estado `accepted`, escuta gravada, intenção de relay) em vez de `pending_identity / release_missing`, sem inventar álbum.

## Pontos de projeto (decidir no código e documentar em comentário e no guia)
1. **Sem explosão de ocorrências/gravações:** reaproveitar uma ocorrência sem lançamento (e sua gravação) existente quando o título e os créditos de artistas coincidem por alias exato (mesma normalização usada pela resolução de ocorrência); sem fusão por similaridade; múltiplas candidatas continuam `pending_identity` como hoje; MBID de gravação, quando presente, identifica a gravação.
2. **Paridade com o legado:** descubra o que o escritor legado cria para escuta sem álbum (`catalog.ResolveListenTrack`) e mantenha a mesma contagem por perfil no lote do ensaio.
3. **Leitores:** o ranking por lançamento (135) deve mostrar o balde "Lançamento desconhecido"; a soma por lançamento continua igual ao total por gravação; páginas de faixa funcionam sem álbum (já tratado na 135).
4. **Inbox (184):** a escuta sem álbum fica classificada como "incompleta" (não "faltante").
5. **Exportação e reimportação (183):** preservam a ocorrência sem lançamento (já suportado; confirmar com teste).
6. **Relay:** segue o mesmo caminho das escutas aceitas.
7. **Continuam rejeitadas:** escuta sem artista ou sem título (400 nos dois caminhos).

## Restrições
- Não mudar a política de retry, a fila de revisão (181), o descarte (185) nem o timeout do MusicBrainz (reagenda).
- Migração só se necessária, aditiva (schema atual v20), com guarda de downgrade.
- Testes por classe: sem álbum aceito; mesma faixa repetida reutiliza a ocorrência; duas faixas diferentes sem álbum; mesmo título com artistas diferentes; ambiguidade; sem artista/título rejeitada; reenvio no mesmo segundo (Revisar ligado e desligado); ranking por lançamento com o balde; Inbox; exportação e reimportação; relay; paridade de contagem com o legado no lote sintético do ensaio (`scripts/rehearse_issue172.py` na branch da 172 pode servir de modelo; não dependa de dados do usuário).
- Docs (guia de ingestão) e CHANGELOG [Unreleased] em inglês.
- Advisor (effort high) obrigatório depois da implementação (identidade); acionado pelo usuário.
