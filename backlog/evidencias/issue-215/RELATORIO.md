# ISSUE-215 — Medição: MusicBrainz por nome × ListenBrainz lookup para achar o MBID de gravação

Rodada de 26/11/2026. Medição com dados reais, executada pelo script
`medir_issue_215.py` (neste mesmo diretório). Nenhum código de produção foi
alterado.

## 1. O que foi medido

Para cada escuta da amostra, o MBID de referência é o `recording_mbid` enviado
pelo cliente (`track_metadata.additional_info.recording_mbid`); o MBID do
mapeador do próprio ListenBrainz (`mbid_mapping`) nunca foi usado como
referência. O MBID é então descartado da consulta e as duas fontes são
perguntadas por nome, duração e álbum:

- **MusicBrainz:** `GET /ws/2/recording?query=recording:"<faixa>" AND artist:"<artista>"&fmt=json&limit=100`,
  com candidatos conferidos por duração (`length` em ms), álbum (título de
  release) e créditos de artista.
- **ListenBrainz:** `GET /1/metadata/lookup/?artist_name=&recording_name=&release_name=`,
  com token (lido do arquivo, nunca gravado nem impresso).

Desfecho por fonte: **acertou** (MBID igual à referência), **errou** (falso
positivo), **ambíguo** (mais de um candidato plausível), **ausente** (nenhum
candidato) e **erro HTTP**. Limites respeitados: no máximo 1 requisição por
segundo no MusicBrainz; no ListenBrainz, cabeçalhos `X-RateLimit-*` observados
e espera antes de continuar em 429/503. `User-Agent` nas duas fontes:
`TuneTrail/0.76 ( https://github.com/ebrgr/TuneTrail )`.

## 2. Amostra

| Métrica | Valor |
|---|---|
| Alvo do despacho | 300 escutas |
| Amostra sorteada | 35 |
| Escutas varridas do histórico | 1299 (fim do histórico) |
| Escutas com `recording_mbid` do cliente | 39 |
| Escutas por cliente (`submission_client`) | navidrome 166 · listenbrainz 1133 |
| Escutas do Navidrome com `recording_mbid` | 39 de 166 |
| Escutas do Navidrome sem `recording_mbid` | 127 de 166 (76,5%) |
| Combinações distintas (artista, faixa, álbum) | 35 |
| Combinações excluídas por MBID de referência conflitante | 0 |
| Amostra vinda do Navidrome | 35 (100%) |
| Escutas com duração conhecida | 35 (100%) |
| Escutas com álbum preenchido | 35 (100%) |

As 39 escutas com MBID do cliente são todas do Navidrome; a amostra de 35
combinações veio, portanto, inteira do cliente que importa. O alvo de 300 (ou
a redução para 150) não se aplicou por falta de dados, não por tempo: não há
mais escutas com referência confiável no histórico consultado. Percentuais
sobre uma amostra tão pequena têm intervalo de confiança largo.

## 3. Resultados por fonte (n = 35)

| Fonte | Acertou | Errou | Ambíguo | Ausente | Erro HTTP |
|---|---|---|---|---|---|
| MusicBrainz por nome (regra proposta: score ≥ 95, duração ±5 s, álbum e créditos conferidos) | 27 (77,1%) | 0 (0%) | 6 (17,1%) | 2 (5,7%) | 0 (0%) |
| ListenBrainz lookup | 35 (100%) | 0 (0%) | 0 (0%) | 0 (0%) | 0 (0%) |

Sem nenhuma regra (apenas o 1º resultado do MusicBrainz por score), o acerto
cai para 10/35 (28,6%) — o score sozinho não distingue versões, como já
suspeitava o doc-048. O acerto de 100% do ListenBrainz vem do mapeador do
próprio ListenBrainz (o endpoint consultado é o lookup do serviço), o que
possivelmente favorece essa fonte; não é evidência de superioridade.

As 35 consultas do MusicBrainz foram respondidas (0 erros HTTP). O script repete
a consulta em 429 e 503, mas não registra quantas tentativas cada uma precisou,
então não há dado sobre repetições internas.

## 4. Varredura de regras do MusicBrainz (álbum e créditos ligados, n = 35)

| Tolerância (s) | Limiar de score | Avaliadas | Decisivas | Acertos | Falsos | Ambíguas | Ambíguas c/ referência | Ausentes | Precisão | Cobertura |
|---|---|---|---|---|---|---|---|---|---|---|
| 2 | 90 | 35 | 31 | 30 | 1 | 1 | 1 | 3 | 96,8% | 85,7% |
| 2 | 95 | 35 | 31 | 30 | 1 | 1 | 1 | 3 | 96,8% | 85,7% |
| 2 | 100 | 35 | 31 | 30 | 1 | 1 | 1 | 3 | 96,8% | 85,7% |
| 3 | 90 | 35 | 30 | 29 | 1 | 2 | 2 | 3 | 96,7% | 82,9% |
| 3 | 95 | 35 | 30 | 29 | 1 | 2 | 2 | 3 | 96,7% | 82,9% |
| 3 | 100 | 35 | 30 | 29 | 1 | 2 | 2 | 3 | 96,7% | 82,9% |
| 5 | 90 | 35 | 27 | 27 | 0 | 6 | 5 | 2 | 100% | 77,1% |
| 5 | 95 | 35 | 27 | 27 | 0 | 6 | 5 | 2 | 100% | 77,1% |
| 5 | 100 | 35 | 27 | 27 | 0 | 6 | 5 | 2 | 100% | 77,1% |
| 10 | 90 | 35 | 25 | 25 | 0 | 8 | 7 | 2 | 100% | 71,4% |
| 10 | 95 | 35 | 26 | 26 | 0 | 7 | 6 | 2 | 100% | 74,3% |
| 10 | 100 | 35 | 26 | 26 | 0 | 7 | 6 | 2 | 100% | 74,3% |

Precisão = acertos / decisivas (decisivas = acertos + falsos). Cobertura =
acertos / avaliadas. O limiar de score quase não muda o desfecho (90, 95 e 100
são equivalentes em cada tolerância); a tolerância é o fator que decide. Com
±2 ou ±3 s, uma escuta resolve para um **único candidato errado** (outra
versão com duração quase idêntica); com ±5 s o candidato correto também entra
e o caso vira ambíguo — que a regra rejeita e manda para a próxima fonte, em
vez de gravar um MBID errado. Entre as duas tolerâncias, **6 escutas mudam de
desfecho**; a que decide a regra é **1** (falso positivo em ±2 s, ambígua em
±5 s).

## 5. Efeito de desligar álbum e créditos (na regra proposta: 5 s / 95)

| Álbum conferido | Créditos conferidos | Decisivas | Acertos | Falsos | Ambíguas | Ausentes | Precisão | Cobertura |
|---|---|---|---|---|---|---|---|---|
| sim | sim | 27 | 27 | 0 | 6 | 2 | 100% | 77,1% |
| sim | não | 27 | 27 | 0 | 6 | 2 | 100% | 77,1% |
| não | sim | 16 | 14 | 2 | 19 | 0 | 87,5% | 40,0% |
| não | não | 16 | 14 | 2 | 19 | 0 | 87,5% | 40,0% |

Desligar a conferência de álbum derruba a cobertura de 77,1% para 40,0% e
deixa entrar 2 falsos positivos. A conferência de créditos de artista não
mudou nenhum desfecho nesta amostra (todo candidato que passava nas demais
conferências já tinha o crédito correto); é barata e fica mantida por
segurança.

## 6. Cruzamento das fontes (regra proposta no MusicBrainz)

| MusicBrainz \ ListenBrainz | LB acertou | LB errou | LB ausente | LB erro |
|---|---|---|---|---|
| MB acertou | 27 | 0 | 0 | 0 |
| MB errou | 0 | 0 | 0 | 0 |
| MB ambíguo | 6 | 0 | 0 | 0 |
| MB ausente | 2 | 0 | 0 | 0 |
| MB erro | 0 | 0 | 0 | 0 |

Na amostra, o ListenBrainz devolveu o MBID de referência para as 8 escutas que
o MusicBrainz não resolveu (6 ambíguas e 2 ausentes) e não houve falso
positivo em nenhuma das duas fontes. O MBID correto está entre os candidatos
aprovados do MusicBrainz em 32/35 escutas (91,4%: 27 acertos + 5 ambíguas com
a referência). O lookup do ListenBrainz devolve o MBID do mapeador do próprio
serviço, então essa comparação mostra o que cada fonte devolve na amostra,
não qual é melhor em geral.

## 7. Tempo por consulta (latência da requisição HTTP, em ms)

| Fonte | n | Mínimo | Mediana | p95 | Máximo |
|---|---|---|---|---|---|
| MusicBrainz busca | 35 | 784 | 1346 | 1992 | 2771 |
| ListenBrainz lookup | 35 | 878 | 958 | 1032 | 1119 |
| MusicBrainz ISRC | 49 | 744 | 839 | 1220 | 1498 |

## 8. Ponte por ISRC (subamostra de 35 MBIDs de referência)

O ISRC enviado pelo Navidrome não estava disponível nesta rodada; a ponte foi
medida com os ISRCs que o próprio MusicBrainz associa à gravação de
referência (`GET /ws/2/recording/<mbid>?inc=isrcs&fmt=json` e depois
`GET /ws/2/isrc/<isrc>?fmt=json`).

| Métrica | Valor |
|---|---|
| Gravações na subamostra | 35 |
| Com ao menos 1 ISRC | 35 (100%) |
| Sem ISRC | 0 (0%) |
| Erro ao buscar gravação | 0 |
| Total de ISRCs | 49 |
| Gravações com mais de 10 ISRCs (truncadas) | 0 |
| Consultas de ISRC válidas | 49 (0 com erro) |
| ISRCs que levam a uma única gravação | 47 (95,9% das válidas) |
| ISRCs cujo destino inclui a gravação de referência | 49 (100% das válidas) |
| ISRCs ambíguos que incluem a referência | 2 |
| Gravações que ganham MBID sem ambiguidade via ISRC | 35 (100% das com ISRC) |
| Gravações cujos ISRCs levam só a uma gravação errada | 0 (0% das com ISRC) |

**O que vale nesta tabela é a unicidade (47 de 49 ISRCs, 95,9%).** As linhas
de 100% são tautológicas: os ISRCs saem da própria gravação de referência, e
«ganham MBID sem ambiguidade» conta gravações com ao menos um ISRC não
ambíguo — uma escuta traz um único ISRC, não vários. A métrica não mede ISRC
vindo do Navidrome.

## 9. Regra de aceitação proposta (MusicBrainz por nome)

Campos conferidos: **nome de gravação e artista na consulta**; entre os
candidatos, **álbum (título de release), créditos de artista e duração**.

- Limiar de score: **≥ 95** (90, 95 e 100 produzem o mesmo desfecho na
  amostra; 95 deixa margem).
- Tolerância de duração: **±5 s** sobre o `length` em ms, quando a escuta tem
  duração conhecida. A tolerância serve para **afastar falsos positivos, não
  para ser generosa**: com ±2 s, um candidato errado (outra versão com duração
  quase idêntica) vence sozinho; com ±5 s ele vira ambíguo e é rejeitado.
  Sem duração conhecida na escuta, o filtro de duração não se aplica e o
  candidato precisa de álbum e créditos.
- Exatamente **1 candidato** que satisfaz a regra → aceita. Mais de 1 →
  ambíguo, não aceita, segue para a próxima fonte. Nenhum → ausente, segue.
- Candidato com `length` desconhecido não passa no filtro de duração (não
  confia no que não dá para conferir).

**Caveat da seleção dentro da amostra:** a regra foi escolhida pelo máximo de
precisão **medido nesta mesma amostra** de 35 escutas (12 combinações de
tolerância e limiar, com álbum e créditos ligados; empate decidido por
cobertura e depois por limiar maior). Os 100% de precisão da regra proposta
são portanto um teto dentro da amostra, **não desempenho esperado em
produção**: com ±2 s a precisão seria 96,8% e a cobertura 85,7% (aceitando 1
falso positivo), e apenas **6 escutas separam o desfecho das duas
tolerâncias**, sendo **1** a escuta que decide (falso positivo em ±2 s,
ambígua em ±5 s). Trate todos os números deste relatório como ordem de
grandeza e direção, não como estimativa fina.

Justificativa da escolha: um MBID errado vale menos que nenhum (o desenho do
doc-048 diz que ambíguo não é aceito), então a preferência por precisão sobre
cobertura segue o princípio do desenho; os números apenas mostram em que
ponto da amostra essa troca acontece.

## 10. Ordem recomendada das fontes (cadeia de identidade)

1. **Navidrome** (quando configurado): MBID direto e, quando o cliente enviar,
   **ISRC**. A consulta por ISRC é promissora pela unicidade medida (95,9%
   dos ISRCs levam a uma única gravação), mas essa evidência **não vem de
   ISRC do Navidrome** — vem dos ISRCs que o próprio MusicBrainz associa à
   gravação de referência, e não foi medida com ISRC do cliente.
2. **MusicBrainz por nome** com a regra da seção 9 (aberto, sem token).
3. **ListenBrainz `/1/metadata/lookup/`** (exige token do relay).

A ordem segue a decisão do doc-048 de **serviços abertos antes dos que exigem
conta** — não os números desta medição. Com n = 35 e o viés da amostra
(seção 11), os dados **não sustentam que um serviço é melhor que o outro**:
o ListenBrainz acertou mais na amostra, mas devolve o MBID do mapeador do
próprio serviço, e o MusicBrainz, com a regra, errou zero.

## 11. Limitações e o que não pôde ser medido

- **Amostra menor que o alvo:** a conta tem 1299 escutas, 39 com MBID do
  cliente (35 combinações). Não houve redução por tempo; simplesmente não há
  mais escutas com referência confiável no histórico consultado.
- **Viés de seleção (Risco 4):** a amostra só inclui escutas que **já trazem**
  MBID do cliente — faixas bem etiquetadas, que o MusicBrainz conhece bem. A
  população que a ISSUE-216 quer atender são as escutas **sem** MBID: pelas
  contagens medidas, **127 das 166 escutas do Navidrome (76,5%) não têm
  `recording_mbid`**. Os acertos de 77,1% (MusicBrainz) e 100% (ListenBrainz)
  provavelmente superestimam o desempenho nessas faixas.
- **ISRC medido por construção (Risco 5):** os ISRCs saem da gravação de
  referência no próprio MusicBrainz; as linhas de 100% da seção 8 são
  esperadas por construção e a métrica que vale é a de unicidade (47/49,
  95,9%). A recomendação de consultar o ISRC antes da busca por nome não se
  apoia em ISRC vindo do Navidrome, que não estava disponível nesta rodada.
- **Ordem das fontes não é conclusão dos dados (Risco 6):** a ordem
  recomendada segue a decisão do doc-048 (abertos antes de contas), não os
  números; o lookup do ListenBrainz devolve o MBID do mapeador do próprio
  serviço, o que possivelmente favorece essa fonte.
- **Tentativas não registradas:** o script repete a consulta em 429 e 503, mas
  não registra quantas tentativas cada consulta precisou. A rodada terminou com
  0 erros HTTP nas 35 buscas do MusicBrainz e nas 35 do ListenBrainz, e a
  latência do MusicBrainz não mostra cauda lenta; o script lê os cabeçalhos
  `X-RateLimit-*` do ListenBrainz.
- Percentuais sobre n = 35 têm intervalo de confiança largo; os números valem
  como ordem de grandeza e direção, não como estimativa fina.

## 12. Reprodução

```
python3 backlog/evidencias/issue-215/medir_issue_215.py --phase all \
  --target 300 --isrc-size 100 --seed 20261126
```

O script lê o token de `~/.cache/tunetrail-215/lb_token` (ou de
`ISSUE215_LB_TOKEN_FILE`, com `expanduser`), nunca o imprime nem o grava, e
mantém os dados brutos apenas em `/tmp/issue215-raw` (apagados ao final). A
semente fixa (`20261126`) torna a amostra reprodutível.
