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
| Combinações distintas (artista, faixa, álbum) | 35 |
| Combinações excluídas por MBID de referência conflitante | 0 |
| Escutas vindas do Navidrome | 0 |
| Escutas com duração conhecida | 35 (100%) |
| Escutas com álbum preenchido | 35 (100%) |

O alvo de 300 (ou a redução para 150) não se aplicou por falta de dados, não
por tempo: a conta consultada tem 1299 escutas no total e apenas 39 trazem
MBID enviado pelo cliente. A medição usou as 35 combinações distintas
existentes. Percentuais sobre uma amostra tão pequena têm intervalo de
confiança largo.

## 3. Resultados por fonte (n = 35)

| Fonte | Acertou | Errou | Ambíguo | Ausente | Erro HTTP |
|---|---|---|---|---|---|
| MusicBrainz por nome (regra proposta: score ≥ 95, duração ±5 s, álbum e créditos conferidos) | 28 (80,0%) | 0 (0%) | 6 (17,1%) | 1 (2,9%) | 0 (0%) |
| ListenBrainz lookup | 35 (100%) | 0 (0%) | 0 (0%) | 0 (0%) | 0 (0%) |

Sem nenhuma regra (apenas o 1º resultado do MusicBrainz por score), o acerto
cai para 11/35 (31,4%) — o score sozinho não distingue versões, como já
suspeitava o doc-048.

Durante a janela da medição o MusicBrainz respondeu 503 de forma intermitente;
2 consultas falharam na primeira passada e foram repetidas com espera até
obter resposta. Na base final, as 35 consultas do MusicBrainz foram
respondidas (0 erros HTTP).

## 4. Varredura de regras do MusicBrainz (álbum e créditos ligados, n = 35)

| Tolerância (s) | Limiar de score | Avaliadas | Decisivas | Acertos | Falsos | Ambíguas | Ambíguas c/ referência | Ausentes | Precisão | Cobertura |
|---|---|---|---|---|---|---|---|---|---|---|
| 2 | 90 | 35 | 32 | 31 | 1 | 1 | 1 | 2 | 96,9% | 88,6% |
| 2 | 95 | 35 | 32 | 31 | 1 | 1 | 1 | 2 | 96,9% | 88,6% |
| 2 | 100 | 35 | 31 | 30 | 1 | 1 | 1 | 3 | 96,8% | 85,7% |
| 3 | 90 | 35 | 31 | 30 | 1 | 2 | 2 | 2 | 96,8% | 85,7% |
| 3 | 95 | 35 | 31 | 30 | 1 | 2 | 2 | 2 | 96,8% | 85,7% |
| 3 | 100 | 35 | 30 | 29 | 1 | 2 | 2 | 3 | 96,7% | 82,9% |
| 5 | 90 | 35 | 28 | 28 | 0 | 6 | 5 | 1 | 100% | 80,0% |
| 5 | 95 | 35 | 28 | 28 | 0 | 6 | 5 | 1 | 100% | 80,0% |
| 5 | 100 | 35 | 27 | 27 | 0 | 6 | 5 | 2 | 100% | 77,1% |
| 10 | 90 | 35 | 26 | 26 | 0 | 8 | 7 | 1 | 100% | 74,3% |
| 10 | 95 | 35 | 27 | 27 | 0 | 7 | 6 | 1 | 100% | 77,1% |
| 10 | 100 | 35 | 26 | 26 | 0 | 7 | 6 | 2 | 100% | 74,3% |

Precisão = acertos / decisivas (decisivas = acertos + falsos). Cobertura =
acertos / avaliadas. O limiar de score quase não muda o desfecho (90 e 95 são
equivalentes na amostra); a tolerância é o fator que decide. Com ±2 ou ±3 s,
uma escuta resolve para um **único candidato errado** (outra versão com
duração quase idêntica); com ±5 s o candidato correto também entra e o caso
vira ambíguo — que a regra rejeita e manda para a próxima fonte, em vez de
gravar um MBID errado. A tolerância maior só aumenta a ambiguidade, nunca os
falsos positivos.

## 5. Efeito de desligar álbum e créditos (na regra proposta: 5 s / 95)

| Álbum conferido | Créditos conferidos | Decisivas | Acertos | Falsos | Ambíguas | Ausentes | Precisão | Cobertura |
|---|---|---|---|---|---|---|---|---|
| sim | sim | 28 | 28 | 0 | 6 | 1 | 100% | 80,0% |
| sim | não | 28 | 28 | 0 | 6 | 1 | 100% | 80,0% |
| não | sim | 15 | 14 | 1 | 20 | 0 | 93,3% | 40,0% |
| não | não | 15 | 14 | 1 | 20 | 0 | 93,3% | 40,0% |

Desligar a conferência de álbum derruba a cobertura de 80,0% para 40,0% e
deixa entrar um falso positivo. A conferência de créditos de artista não
mudou nenhum desfecho nesta amostra (todo candidato que passava nas demais
conferências já tinha o crédito correto); é barata e fica mantida por
segurança.

## 6. Cruzamento das fontes (regra proposta no MusicBrainz)

| MusicBrainz \ ListenBrainz | LB acertou | LB errou | LB ausente | LB erro |
|---|---|---|---|---|
| MB acertou | 28 | 0 | 0 | 0 |
| MB errou | 0 | 0 | 0 | 0 |
| MB ambíguo | 6 | 0 | 0 | 0 |
| MB ausente | 1 | 0 | 0 | 0 |
| MB erro | 0 | 0 | 0 | 0 |

O ListenBrainz recuperou as 7 escutas que o MusicBrainz não resolveu (6
ambíguas e 1 ausente). As duas fontes não se contradizem nesta amostra:
nenhum falso positivo em nenhuma delas. No total da amostra, o MBID correto
está entre os candidatos aprovados do MusicBrainz em 33/35 escutas (94,3%:
28 acertos + 5 ambíguas com a referência).

## 7. Tempo por consulta (latência da requisição HTTP, em ms)

| Fonte | n | Mínimo | Mediana | p95 | Máximo |
|---|---|---|---|---|---|
| MusicBrainz busca | 35 | 739 | 1356 | 19828 | 28700 |
| ListenBrainz lookup | 35 | 814 | 971 | 1080 | 1239 |
| MusicBrainz ISRC | 49 | 737 | 816 | 1230 | 1596 |

A cauda do MusicBrainz (p95 ≈ 19,8 s) reflete a instabilidade do serviço na
janela da medição (respostas lentas sob carga, com 503 intermitentes); a
mediana (1,4 s) representa o caso normal.

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

Na subamostra, todo ISRC leva à gravação de referência e 95,9% levam a uma
única gravação: a ponte por ISRC é determinística, mas a medição usou ISRCs do
próprio MusicBrainz — ISRCs vindos do cliente podem ter grafia diferente e não
foram medidos.

## 9. Regra de aceitação proposta (MusicBrainz por nome)

Campos conferidos: **nome de gravação e artista na consulta**; entre os
candidatos, **álbum (título de release), créditos de artista e duração**.

- Limiar de score: **≥ 95** (90 produz o mesmo desfecho na amostra; 95 deixa
  margem).
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

Justificativa: um MBID errado vale menos que nenhum (o desenho do doc-048 diz
que ambíguo não é aceito). Com a regra, a amostra teve 100% de precisão e
80,0% de cobertura; a alternativa de ±2 s troca precisão (96,9%) por
cobertura (88,6%), aceitando 1 falso positivo.

## 10. Ordem recomendada das fontes (cadeia de identidade)

1. **Navidrome** (quando configurado): MBID direto e, quando o cliente enviar,
   **ISRC** — a consulta por ISRC deve anteceder a busca por nome, pois na
   subamostra 95,9% dos ISRCs levam a uma única gravação e todos incluíam a
   referência.
2. **MusicBrainz por nome** com a regra da seção 9 (aberto, sem token):
   cobre 80,0% da amostra sem nenhum falso positivo; em 94,3% o MBID correto
   está entre os candidatos aprovados.
3. **ListenBrainz `/1/metadata/lookup/`** (exige token do relay): recuperou as
   7 escutas que o MusicBrainz não resolveu, fechando 100% da amostra sem
   nenhum falso positivo.

Serviços abertos antes dos que exigem conta, como decidido no doc-048; o
ListenBrainz fica como complemento, não como primeira fonte.

## 11. Lacunas e o que não pôde ser medido

- **Amostra menor que o alvo:** a conta tem 1299 escutas, 39 com MBID do
  cliente (35 combinações). Não houve redução por tempo; simplesmente não há
  mais escutas com referência confiável no histórico consultado.
- **Nenhuma escuta vinda do Navidrome** nas 1299 varridas: a preferência por
  `submission_client = navidrome` não pôde ser aplicada; a amostra usa os
  demais clientes.
- **ISRC do Navidrome indisponível nesta rodada:** a ponte foi medida com
  ISRCs do próprio MusicBrainz, o que mede a qualidade da ponte, não a
  disponibilidade de ISRC no cliente.
- **Instabilidade do MusicBrainz durante a janela:** 503 intermitentes;
  2 consultas falharam na primeira passada e foram repetidas até obter
  resposta (nenhum erro HTTP na base final).
- Nenhum 429/503 do ListenBrainz ocorreu nesta rodada; os cabeçalhos
  `X-RateLimit-*` foram respeitados mesmo assim.
- Percentuais sobre n = 35 têm intervalo de confiança largo; os números valem
  como ordem de grandeza e direção, não como estimativa fina.

## 12. Reprodução

```
python3 backlog/evidencias/issue-215/medir_issue_215.py --phase all \
  --target 300 --isrc-size 100 --seed 20261126
```

O script lê o token de `/Users/evertonberger/.cache/tunetrail-215/lb_token`
(ou de `ISSUE215_LB_TOKEN_FILE`), nunca o imprime nem o grava, e mantém os
dados brutos apenas em `/tmp/issue215-raw` (apagados ao final). A semente
fixa (`20261126`) torna a amostra reprodutível.
