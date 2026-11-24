# ISSUE-175 — Lookup ListenBrainz para MBIDs ausentes

O trabalhador de enriquecimento da ISSUE-174 reserva o recibo já resolvido e usa
Subsonic quando configurado. A etapa ListenBrainz verifica os campos restantes
por gravação e por crédito de artista. Considera o snapshot, o catálogo, as
declarações existentes no perfil e o resultado Subsonic da mesma reserva.

O token deve pertencer a um relay aprovado para
`https://api.listenbrainz.org/1` no perfil do recibo. A geração registrada na
admissão precisa continuar ativa e o destino precisa estar disponível no
relay. A consulta usa HTTPS público sem proxy ou redirecionamento. A remoção,
troca ou suspensão do relay impede que o trabalhador use a configuração para
novas consultas. A gravação também revalida a versão da configuração e a
evidência local após a resposta.

## Contrato verificado

- GET `/1/metadata/lookup/` exige `Authorization: Token <user token>`. O
  servidor valida a credencial e usa os nomes de artista, faixa e álbum
  opcional para o mapeamento. O código do servidor fixa 250 caracteres para a
  soma desses campos e 50 itens na variante POST. A implementação usa uma
  consulta GET por tentativa e `metadata=true&inc=artist` quando precisa de
  créditos nomeados. [API de metadados](https://listenbrainz.readthedocs.io/en/latest/users/api/metadata.html)
  · [Implementação do lookup](https://github.com/metabrainz/listenbrainz-server/blob/57d62788b1b3bd389cfce81e6aecdd39a91126f6/listenbrainz/webserver/views/metadata_api.py)
- A documentação pede no máximo uma chamada por segundo e orienta o uso dos
  cabeçalhos de limite. O trabalhador espaça as chamadas de lookup em um
  segundo entre perfis e respeita o período indicado quando o saldo se esgota
  ou o servidor devolve HTTP 429. Uma falha de rede termina a tentativa. O
  cache guarda até 1.024 respostas válidas por dez minutos, com chaves por
  perfil, geração e consulta. [Autenticação e limites da API](https://listenbrainz.readthedocs.io/en/latest/users/api/#rate-limiting)
- O endpoint oferece um candidato, sem lista de alternativas ou indicador de
  confiança. TuneTrail exige os nomes correspondentes e confirma IDs de
  gravação, edição e artistas quando disponíveis. Para gravação ainda sem ID,
  exige nome do álbum. Uma resposta com a mesma combinação textual em
  gravações indistinguíveis permanece sujeita à escolha canônica do provedor.
  Para artistas, exige um crédito nomeado único ligado ao resultado e só
  aceita o crédito agregado se houver um único artista. [Fluxo do mapeador](https://github.com/metabrainz/listenbrainz-server/blob/57d62788b1b3bd389cfce81e6aecdd39a91126f6/listenbrainz/webserver/views/metadata_api.py#L247-L342)

## Persistência

O schema 24 guarda a declaração `listenbrainz-lookup` com data, perfil e escuta
em `metadata_values`. O snapshot original e os IDs do catálogo são preservados.
O v2 exporta e reimporta as declarações e seus conflitos como evidência da
escuta. Os consumidores selecionam declarações unívocas dentro do próprio
perfil. O v1 omite valores que exigem a procedência do v2.

O gatilho da ISSUE-174 enfileira recibos com Subsonic ativo. Quando Subsonic
está desligado, a decisão automática ou manual só enfileira o recibo se
existir relay oficial elegível. A sondagem ociosa usa apenas leitura. O lease e
o teto de reservas da ISSUE-174 limitam a recuperação após interrupção.
