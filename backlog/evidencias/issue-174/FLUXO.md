# ISSUE-174 — fluxo e decisões de implementação

## Avaliação e consulta

1. A resolução da escuta termina normalmente. Apenas recibos ListenBrainz aceitos ou duplicados, enquanto o Subsonic está configurado, criam trabalho de enriquecimento na mesma transação; recibos históricos não são varridos.
2. O trabalhador próprio reserva o recibo com lease e limite de duas reservas, incluindo recuperações após interrupção. Serviço desligado não cria trabalho novo; ao desligar, o trabalho anterior pendente é encerrado. Falha externa não altera a decisão de ingestão nem provoca repetição infinita.
3. O pedido é filtrado por entidade: IDs recebidos ou já conhecidos no escopo do perfil e da instância não são substituídos. Créditos com IDs parciais sem associação inequívoca não são atribuídos por posição.
4. O provedor procura pelo título via `search3`, requer correspondência de título, artista e álbum, confronta duração, posição e IDs disponíveis e recusa múltiplos candidatos ou uma página possivelmente truncada. A consulta de artista usa o ID nativo do artista ligado à música, com conferência do nome. O MBID da música requer `mediaType=song`.
5. As chamadas usam POST, prazo, limite de taxa e cache limitado por perfil e configuração. Credenciais não são registradas nem incluídas nos arquivos de exportação.

## Gravação e consumo

Depois da rede, uma transação curta revalida o recibo, perfil, versão, ocorrência, gravação, créditos e metadados do catálogo. Um resultado antigo é descartado; o snapshot original nunca é reescrito. MBIDs externos são declarações com fonte e instante, sem promoção a `validated` nem substituição de IDs existentes. Valores conflitantes ficam preservados e não se escolhe um deles automaticamente.

As declarações são privadas ao perfil. Na exclusão de escuta, uma declaração de artista é removida quando não há escuta relacionada; uma declaração de gravação acompanha outra ocorrência da mesma gravação se o perfil ainda a escuta, e é removida quando não resta escuta relacionada. Exclusão do perfil remove suas declarações. Apagar apenas o recibo não apaga metadados ainda utilizados.

A exportação v2 reúne os dados em três consultas por página (gravações, artistas e recibos), além das consultas de catálogo já existentes. Ela preserva inclusive declarações conflitantes, fonte e data; usa ordinais para distinguir instâncias sem revelar o hash de credenciais. IDs provenientes de declarações ficam em campos próprios, para que leitores antigos que ignorem a procedência não os promovam a identidade global. A importação v2 valida a procedência e restaura as declarações apenas no perfil de destino. O formato v1 omite essas declarações, pois não representa procedência.

## Tabelas e estados

- `enrichment_configuration`: uma linha que habilita ou desabilita a criação de trabalho; evita fila e escrita contínua quando não há provedor configurado.
- `enrichment_jobs`: um trabalho por recibo, com lease e tentativas limitadas. `pending` e `working` representam trabalho; `found`, `absent`, `ambiguous`, `disabled` e `failed` registram resultados terminais distintos, sem virar pendência de ingestão.
- `enrichment_recording_claims`: declarações por perfil, ocorrência, instância, MBID e fonte. A ocorrência permite acompanhar reconciliações e exclusões sem associar a declaração globalmente.
- `enrichment_artist_claims`: declarações por perfil, artista, instância, MBID e fonte. A tabela separada mantém chaves estrangeiras reais e a política de retenção própria dos artistas.

A migração 025 preserva as declarações da 024 e remove a dependência do recibo. As tabelas de declarações permanecem separadas para garantir integridade referencial; a fila é independente para não vincular falhas de rede à recuperação de ingestão. Nenhuma tabela guarda credenciais ou resposta bruta do provedor.

A retenção tem dois gatilhos porque excluir uma escuta e excluir a própria ocorrência seguem caminhos diferentes de cascata no SQLite; a transferência antes da exclusão da ocorrência evita perder a declaração antes de localizar outra edição. Os testes cobrem os dois caminhos e a remoção do perfil.
