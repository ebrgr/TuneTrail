---
id: doc-036
title: Estudo da ISSUE-116 — identidade de faixas
type: other
created_date: '2026-11-15T16:00:00Z'
updated_date: '2026-11-15T16:00:00Z'
tags: [estudo]
---
**Tarefa:** ISSUE-116 · **Base examinada:** `097885c`, branch `spike/issue-116-estudo-identidade-de-faixas-entre` · **Data:** 15/11/2026.

Estudo sem implementação. Recomendações aguardam aceite; nenhuma decisão de arquitetura foi aceita, tarefa de implementação criada ou versão publicada. Origem: doc-034 (ISSUE-114); formato de referência: doc-033. A worktree começou com uma alteração na ISSUE-116 feita pelo coordenador, preservada neste trabalho.

## Recomendação

Separar **gravação** de **ocorrência da faixa em um lançamento** e fazer cada escuta apontar para a ocorrência que foi ouvida. Isso permite reunir a mesma gravação no ranking de faixas e conservar o álbum correto no histórico. Corrigir antes a consulta de aliases e a aceitação de MBIDs conflitantes; preservar grafia original e oferecer agrupamentos reversíveis por perfil para caixa/remaster/deluxe. Não fundir automaticamente históricos por semelhança textual.

Manter o modelo atual é uma alternativa de menor custo, mas apenas corrigir aliases não resolve a perda do álbum com MBID nem os títulos iguais. Acrescentar uma relação N:N entre faixa e álbum também é insuficiente se a escuta continuar sem identificar qual ocorrência foi ouvida: um JOIN contaria a mesma escuta em todos os álbuns relacionados.

## Associação atual, com localização

As linhas se referem à base acima; `db/migrations_sqlite` é o esquema SQLite executado, incluindo a migração 003, e não somente o SQL inicial.

| Etapa | Código e comportamento |
| --- | --- |
| Entrada e perfil | `internal/catalog/catalog.go:94`: serializa a associação; `:102` resolve ProfileID e `:105` retira o escopo de perfil durante a consulta ao catálogo compartilhado. Não há catálogo próprio por perfil. |
| Artista | `internal/catalog/associate_artists.go:30`: prioriza mapa de MBIDs, lista de MBIDs, nomes e parsing do artista; `:199` tenta aliases do MusicBrainz; `:262` busca/cria por nome. `internal/db/sqlite/artist.go:32` consulta todos os aliases, com igualdade exata. O EqualFold de `associate_artists.go:304` evita repetições na lista em memória, mas não torna a busca persistente insensível a caixa. |
| Álbum | `internal/catalog/associate_album.go:28`: MBID de release tem prioridade; `:70` consulta o release no MusicBrainz e pode preencher um álbum sem MBID pelos títulos; `:97` guarda títulos do release group como aliases. O ID persistido é de release, não uma entidade release-group separada. `:177` procura título e primeiro artista; sem álbum, usa o título da faixa. `internal/db/sqlite/album.go:32` consulta título primário e relações de artistas. |
| Créditos | `internal/catalog/catalog.go:155` associa artistas ao álbum como relação derivada; `internal/db/sqlite/track.go:162` grava artist_tracks e não promove a participação a crédito independente de álbum. A migração proposta precisa preservar essa distinção da ISSUE-049. |
| Faixa | `internal/catalog/associate_track.go:24`: exige nome, álbum e artistas; `:46` busca MBID de recording globalmente e devolve o encontrado, sem comparar álbum/artistas. Se não encontra MBID, `:81` busca título + álbum + artistas; `:95` tenta o título do MusicBrainz e `:112` cria a faixa. |
| Consulta por título | `internal/db/sqlite/track.go:111`: consulta tracks_with_title, igualdade exata de título e release_id, e exige presença dos artistas enviados via IN/HAVING. Pode aceitar candidatos com artistas adicionais; não exige igualdade dos conjuntos completos. Não usa duração, disco, posição ou identificador de faixa no release. Se há vários candidatos, não relata ambiguidade nem define prioridade. |
| Persistência da faixa | `internal/db/sqlite/track.go:136`: SaveTrack grava um único release_id, duração, MBID, créditos e alias primário; não existe restrição de unicidade por título/álbum/artista. Duplicatas de título são representáveis no banco, mas o resolvedor não as distingue. |
| Escuta | `internal/catalog/catalog.go:223` grava somente TrackID, instante, perfil, usuário legado e cliente; o álbum encontrado anteriormente não acompanha a escuta. `internal/db/sqlite/listen.go:22` usa ON CONFLICT DO NOTHING para a chave `(profile_id, track_id, listened_at)`. |

Esquema: `db/migrations_sqlite/001_initial_schema.sql:41` define tracks com `musicbrainz_id UNIQUE` e `release_id NOT NULL`; `:48` define artist_tracks; `:55` track_aliases; `:106` limita tracks_with_title ao alias primário. A migração `002_unique_mbid.sql:8` mantém a unicidade global de MBID de gravação. `003_listening_profiles.sql:52` define a chave atual das escutas por perfil. Não há coluna de álbum, título original ou metadados de origem na escuta.

Estatísticas: `internal/db/sqlite/track.go:490` agrupa escutas por track_id; `:507` filtra álbum pelo release_id da faixa, e `:705` conta faixas distintas pelo mesmo ID. `internal/db/sqlite/scoped_reads.go:16` injeta o filtro de perfil nas leituras de listens; `:39` verifica a relação entre catálogo e histórico do perfil. Alterações globais do catálogo repercutem em todos os perfis, mesmo que as contagens estejam corretamente filtradas.

## Problemas de origem e reprodução no TuneTrail

Fontes primárias consultadas em 15/11/2026; relatos de versões antigas do Koito não provam o comportamento atual do TuneTrail. As conclusões locais abaixo vêm de código e experimentos fictícios. Não foram usados anexos ou dados reais dos autores.

| Origem | Resultado local | Causa e consequência |
| --- | --- | --- |
| [Koito #34](https://github.com/gabehf/Koito/issues/34): distinguir a mesma música em lançamentos diferentes | **Reproduzido com MBID compartilhado**: uma gravação cadastrada no álbum A, associada com álbum B, retorna ID 1 e AlbumID 1. Sem MBID, o álbum participa da busca e há dois IDs. | `associate_track.go:48` busca só recording MBID; a restrição UNIQUE e o único tracks.release_id impedem representar ambas as ocorrências. O log de `catalog.go:221` usa o álbum recebido, embora a escuta seja salva na faixa do primeiro álbum. |
| [Koito #217](https://github.com/gabehf/Koito/issues/217): juntar single e álbum no ranking | **Reproduzido sem MBID**: a mesma faixa em A/B cria IDs 2/3. Depois de MergeTracks(B→A), ouvir B recria ID 3. Com MBID compartilhado, há um ID, mas perde-se o álbum B. | O título é procurado dentro do álbum. `internal/db/sqlite/track.go:642` redireciona escutas e apaga a origem em `:699`, sem persistir uma regra que reconheça a ocorrência apagada. As demandas #34 e #217 precisam de duas dimensões de identidade. |
| [PR #315](https://github.com/gabehf/Koito/pull/315): títulos iguais no mesmo álbum | **Reproduzido**: duas associações de Intro, mesmo álbum e artista, retornam ID 1; SaveTrack direto permite outro Intro (ID 2). Com MBIDs distintos 1001/1002, a segunda associação também retorna ID 1. | Não há identidade de posição nem conflito explícito. Mais grave: `associate_track.go:58` aceita o resultado textual mesmo com MBID diferente; `:74` só registra aviso e `:76` altera o MBID do objeto devolvido para 1002 enquanto o banco continua em 1001. Dois MBIDs não são proteção suficiente hoje. Duração diferente não participa da decisão. |
| [Koito #214](https://github.com/gabehf/Koito/issues/214): Last.fm junta homônimas de álbuns diferentes | **Condicional**: importação JSON real sem MBID produziu 2 faixas/2 álbuns/2 escutas; com recording MBID compartilhado, 1 faixa/1 álbum/2 escutas. O relato de junção somente por título não se reproduziu com álbuns distintos e sem MBID. | `internal/importer/lastfm.go:78` preserva Album.Text (usa Name se vazio); `:95` interpreta track.MBID como recording MBID; `:122` entrega esses campos ao catálogo. Não há cache local por título nesse importador. A junção atual decorre do caminho global de MBID; álbum ausente também retira a distinção textual. |
| [Koito #369](https://github.com/gabehf/Koito/issues/369): alias secundário ignorado | **Reproduzido**: Principal com alias secundário Alternativo; associar Alternativo no mesmo álbum cria ID 2. | tracks_with_title contém só is_primary=1; `track.go:118` não consulta track_aliases. A busca da interface (`track.go:727`) usa aliases, portanto encontrar pela busca não comprova associação correta na ingestão. |
| [Koito #359](https://github.com/gabehf/Koito/issues/359): variações de caixa | **Reproduzido sem MBID**: Faixa/FAIXA criam IDs 1/2; busca de ARTISTA FICTICIO e ALBUM A não encontra as grafias cadastradas. | Igualdade SQL sem normalização nos três resolvedores persistentes. Não basta o EqualFold usado no parsing/listas. Com MBID já cadastrado, a correspondência por ID contorna a diferença textual. |
| [Koito #360](https://github.com/gabehf/Koito/issues/360): opção de reunir remaster/deluxe | **Recurso ausente, separação reproduzida**: Faixa (Remaster) no mesmo álbum e Faixa em Album A (Deluxe) criam IDs próprios sem MBID. | Nenhuma regra de equivalência de versões é aplicada na associação examinada. Se o mesmo MBID for enviado, a junção global ocorre independentemente de preferência, como em #34. Texto Remaster/Deluxe sozinho não comprova identidade da gravação. |

### Protocolo e resultados

Teste temporário `internal/db/sqlite/issue116_study_tmp_test.go`, pacote sqlite, NewInMemory com migrações reais 001–003. Cada cenário usa banco novo, artista fictício, três álbuns, dois perfis e MBIDs sintéticos. A configuração usa t.TempDir e desativa provedores externos; não abre o banco operacional. A importação usa importer.Upload com JSON local fictício e dois instantes distintos, 1000/1001; a faixa com MBID é pré-cadastrada com duração para dispensar MusicBrainz.

Comando executado: `CGO_ENABLED=0 go test ./internal/db/sqlite -run '^TestIssue116Study$' -v -count=1`.

Resultado final: **PASS**, incluindo 34_217, 315, 315_distinct_MBID, 369, 359_360, 214_import/false e 214_import/true. Os asserts verificam o comportamento problemático atual, não a implementação proposta. O teste foi removido antes da verificação geral e não entra no commit. O código completo do experimento está no apêndice para permitir repetir a medição sem versionar testes de produção.

Ocorrências durante a preparação: a primeira versão do experimento falhou com panic em cfg.ArtistSeparators porque não inicializava cfg.Load; corrigido apenas no teste temporário e repetido com sucesso. Uma busca de arquivos de teste do importador retornou erro de glob do zsh (não há arquivos com aquele padrão); não foi falha do aplicativo. Nenhum desses erros foi ignorado como prova positiva. Não se reproduziu coleta real do Spotify nem se testou equivalência acústica: o estudo mede a identidade conforme metadados recebidos.

## Modelo proposto e regras de resolução

Modelo conceitual, nomes sujeitos à implementação:

| Entidade | Responsabilidade e invariantes |
| --- | --- |
| recordings | Identidade da gravação, recording MBID único quando confiável; aliases e créditos de gravação. Sem MBID, registros provisórios podem continuar separados. Não é identidade da composição: versões ao vivo e regravações permanecem distintas. |
| releases | Lançamento identificado por release MBID; preservar os IDs existentes. release-group MBID, quando disponível, é vínculo de agrupamento, não chave para fundir edições automaticamente. |
| release_tracks (ou tracks evoluída) | Ocorrência com ID estável, recording_id, release_id, título/créditos apresentados, duração, disco/posição opcionais e MusicBrainz track ID opcional. Duas ocorrências podem ter o mesmo título ou a mesma gravação no mesmo lançamento. Não impor UNIQUE(recording_id,release_id) nem UNIQUE(title,release_id). |
| listens | Identidade estável do evento, profile_id, occurrence_id, instante e cliente; preservar o metadado observado e a proveniência conforme o contrato de retenção da ISSUE-079/080. Não inferir o álbum de todos os lançamentos da gravação. |
| Grupos por perfil | Relações próprias de equivalência para exibição/ranking, com membros, motivo e versão da regra. Remover o grupo restaura a visão original sem alterar gravações nem escutas. |

Precedência sugerida: ID de ocorrência confiável → recording MBID + contexto do lançamento/posição → aliases exatos e contexto suficiente → candidato normalizado e único → revisão se ambíguo. Nunca substituir um recording MBID conflitante por título. Um MBID novo pode preencher entidade sem MBID somente com ausência de conflito demonstrada e proveniência registrada; dois IDs já diferentes exigem revisão. Um recording MBID reconhece a gravação, mas não decide sozinho a ocorrência.

Buscar todos os aliases, sempre limitados por contexto de artista/lançamento. Se duas faixas compartilham o alias, não escolher a primeira. Metadados pobres sem posição/ID não permitem separar duas Intro iguais com certeza; manter pendência ou entidade provisória, preservando o evento. Duração é sinal auxiliar: pode faltar ou variar entre fontes, então não deve ser chave única nem prova de equivalência.

Para caixa, preservar texto original e título preferido de exibição. Propor chave normalizada versionada com normalização Unicode e case folding, validada com português, japonês e pares como İ/i e ß/ss; não remover acentos nem pontuação como efeito colateral. Colisões viram candidatos, não migração automática de fusão. Não depender somente do NOCASE do SQLite para normalização Unicode.

Para remaster/deluxe, preferência desligada inicialmente e por perfil, com prévia das correspondências. Regras removendo sufixos servem para sugerir grupos; não devem apagar a edição original nem unir gravações com MBIDs incompatíveis. Uma edição deluxe pode incluir novas faixas; só agrupar membros explicitamente associados. A UI de fusão destrutiva da ISSUE-108 precisa distinguir agrupamento, união comprovada de identidade e deduplicação de evento.

## Migração, riscos e estatísticas por perfil

1. Fazer backup e validar restauração em banco fictício/fixture da ISSUE-064 antes da migração; coordenar a numeração com ISSUE-080, sem assumir que o próximo schema será v4. Usar transação e interromper ingestão durante a troca, evitando dois escritores com versões diferentes.
2. Evoluir as faixas existentes para ocorrências, preservando seus IDs, aliases, artistas, release_id, duração e referências. Criar uma gravação para cada faixa antiga; copiar recording MBID e tirar sua unicidade da ocorrência, mantendo-a em recordings. Não fundir registros sem MBID por título. Guardar mapeamento explícito old_track_id → occurrence_id/recording_id.
3. Migrar escutas 1:1, sem DISTINCT, ON CONFLICT IGNORE ou descarte. Perfil, instante e cliente continuam iguais; cada escuta recebe a ocorrência do seu track_id antigo. Se for preciso ID próprio para eventos, adicioná-lo e tratar idempotência no contrato da ISSUE-081, sem usar só gravação+segundo como identidade definitiva. Preservar a compatibilidade da chave antiga durante a transição onde possível.
4. Registrar origem `legacy` e qualidade desconhecida do lançamento histórico. Álbuns que já foram perdidos pela junção por MBID não podem ser reconstruídos a partir desse banco: migrar o álbum conhecido e oferecer reconciliação posterior com fonte/exportação e prévia, sem inventar atribuições. Não é possível garantir correção retrospectiva sem dados adicionais.
5. Atualizar leitura/escrita, exportação, importadores, now-playing, edição/fusão/exclusão e créditos derivados. Manter URLs de faixa por ID antigo e explicitar que a página é uma ocorrência com ligação para a gravação. Evitar cascata de exclusão de um release apagar a gravação usada em outros; limpar apenas órfãos comprovados. Validar aliases primários e FKs ao final.
6. Comparar antes/depois por perfil: número de escutas, clientes, instantes, primeiro/último evento, totais por álbum/artista, contagens de ocorrências e tempo ouvido. Executar foreign_key_check/integrity_check e testar falha no meio da migração. Rollback operacional restaura backup com a versão anterior; não prometer downgrade SQL sem perda depois de novas escritas.

**Semântica estatística recomendada:** ranking de faixas por gravação; página de álbum e histórico por ocorrência. Exibir “gravações” e “faixas do lançamento” com rótulos claros e validar a escolha com o usuário. Cada evento entra uma vez no ranking da gravação, mesmo se ela aparecer em vários álbuns. Uma escuta A/B da mesma gravação soma 2 na gravação e 1 em cada álbum. Sem MBID/equivalência comprovada, as duas gravações provisórias permanecem separadas até revisão.

Primeira escuta/novidade por gravação usa MIN dos eventos do perfil após resolver a identidade; por ocorrência conserva a estreia em cada lançamento. Grupos podem mudar posição/quantidade de entidades, nunca total bruto de escutas. Créditos múltiplos precisam da regra da ISSUE-092 para evitar somar o mesmo evento duas vezes dentro de um grupo de artistas. Tempo ouvido não deve ser recalculado retroativamente por uma duração global que muda; coordenar a duração observada/efetiva com ISSUE-097.

Como catálogo é compartilhado, fundir uma identidade global muda rankings de todos os perfis. Uma preferência de um perfil não deve provocar esse efeito em outro: agrupamentos ficam por perfil. Consultas novas devem manter o escopo de scoped_reads e requireRelated; testar dois perfis com mesma faixa e mesmo segundo, inclusive um sem escutas para o lançamento consultado. Índices candidatos: listens(profile_id, occurrence_id, listened_at), release_tracks(recording_id,release_id), aliases por chave normalizada; medir planos e tempo em histórico sintético grande, sem criar cache/materialização antecipadamente.

**ISSUE-081:** reconhecer a entidade musical não equivale a decidir que dois eventos são duplicados. A chave atual e MergeTracks (`track.go:656`) podem eliminar eventos coincidentes do mesmo perfil; migrar para gravações não autoriza repetir esse descarte em ocorrências diferentes. Deduplicação continua restrita ao perfil, explicável em ingest_events, mantendo repetições legítimas e revisão de ambiguidade. O contrato deve aceitar ID provisório/ocorrência antes de implementar o pipeline; a revisão do Advisor exigida pela 081 permanece para aquela implementação, fora deste estudo sem trabalhadores auxiliares.

**ISSUE-096:** reaproveitar o princípio de grupo reversível por perfil, sem transformar artistas diferentes em um só. A 096 é especificamente de artistas; ampliar grupos a gravações/álbuns exige tarefa própria e não mudança silenciosa de escopo. Resolver um alias comprovado global difere de preferir uma exibição agrupada local.

## Implementações sugeridas, em ordem

Propostas apenas; nenhum registro novo criado no backlog.

| Ordem | Recorte | Critérios de aceite verificáveis |
| --- | --- | --- |
| 1 | Correção de aliases e conflitos de MBID no resolvedor atual | Alias secundário no mesmo contexto reutiliza ID; alias ambíguo produz decisão explícita; MBIDs diferentes em títulos iguais nunca retornam o primeiro com MBID do segundo; retorno e banco concordam. Cobrir artista adicional e ordem dos artistas. |
| 2 | Contrato de identidade e estatística, alinhado a 079/080/081/092 | Fixtures distinguem gravação/ocorrência/evento; decidir álbum ausente, colisão no segundo e revisão; exemplo A/B produz 2 escutas na gravação e 1 em cada álbum, sem vazamento entre dois perfis. Registrar decisão após aceite, antes da migração. |
| 3 | Migração aditiva e compatibilidade de IDs | Migrar banco com múltiplos perfis, aliases, MBIDs nulos e FKs; contagem de eventos idêntica por perfil, sem fusões automáticas; IDs/URLs anteriores válidos; interrupção simulada reverte transação; restauração de backup comprovada. |
| 4 | Ingestão por ocorrência em todas as fontes | Last.fm e ListenBrainz: mesmo MBID em A/B gera uma gravação e duas ocorrências; dois Intro com IDs de ocorrência/posições diferentes permanecem separados; falta de desambiguador preserva evento para revisão. Reimportação idempotente e repetição legítima testadas pela 081. |
| 5 | Consultas, edição/fusão, exclusão e exportação | Rankings e novidades por perfil seguem o contrato; editar/remover um lançamento não muda outro; exportar/reimportar preserva ocorrência/proveniência; duas escutas simultâneas de perfis distintos sobrevivem; planos de consulta medidos em fixtures grandes. |
| 6 | Normalização de caixa com prévia | Preservar grafia; candidatos Unicode determinísticos em versão de regra conhecida; colisões/MBIDs conflitantes não fundidos; ativar/desativar preferência não muda dados brutos nem outro perfil. |
| 7 | Grupos opcionais de remaster/deluxe | Prévia e desfazer por perfil; lançamentos/IDs originais preservados; bônus de deluxe e gravação ao vivo não entram automaticamente; totais brutos invariantes; rankings com/sem grupo testados, em coerência com a 096. |

## Perguntas que dependem do usuário

1. No ranking padrão de faixas, prefere reunir a mesma **gravação** de single/álbum/coletânea (recomendado), mantendo o detalhamento por lançamento, ou mostrar cada ocorrência separada?
2. Para caixa, prefere sugerir agrupamentos para revisão (recomendado inicialmente) ou associar automaticamente candidatos únicos sem conflito de MBID, preservando a grafia original?
3. Remaster/deluxe deve ser apenas agrupamento reversível por perfil, desligado por padrão (recomendado), ou também deseja uma operação separada de fusão global com prévia dos perfis afetados?

## Validação de entrega

Experimentos isolados: PASS, conforme apêndice. A verificação geral passou e a tarefa foi colocada em **Aguardando aceite** pelo comando `make issue.review ISSUE=116 MSG="Verificação: estudo doc-036; reproduções fictícias de identidade PASS; sem alterações de produção; make check aprovado"`. Resultado: **make check aprovado**, com go vet, testes Go, verificações de UI/JS/Python/estilo e **12/12 jornadas E2E**. O teste temporário já estava removido. A revisão documental do diff não identificou pendências de escopo ou ruído; não há implementação que exija parecer do Reviewer. Como só documentação e registro foram alterados, não há reconstrução do grafo de código a executar. Aceite e publicação dependem do usuário/coordenador; este estudo não os executa. A branch já continha o commit `097885c ISSUE-129: Prepara worktrees do Orca e adapta o fluxo de issues`, herdado da base informada pelo coordenador; ele não foi criado nem alterado neste estudo.

## Apêndice — reprodução temporária

Copiar o bloco abaixo para `internal/db/sqlite/issue116_study_tmp_test.go`, executar o comando indicado e remover o arquivo antes de commitar. Requer somente as dependências já usadas pelo repositório; os asserts documentam a base examinada e deverão mudar após as correções propostas.

```go
package sqlite

import (
	"context"
	"fmt"
	"github.com/ebrgr/TuneTrail/internal/catalog"
	"github.com/ebrgr/TuneTrail/internal/cfg"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/importer"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

// TestIssue116Study measures current behavior on fictional, in-memory fixtures.
func TestIssue116Study(t *testing.T) {
	configDir := t.TempDir()
	require.NoError(t, cfg.Load(func(key string) string {
		switch key {
		case cfg.CONFIG_DIR_ENV:
			return configDir
		case cfg.DISABLE_DEEZER_ENV, cfg.DISABLE_COVER_ART_ARCHIVE_ENV, cfg.DISABLE_MUSICBRAINZ_ENV:
			return "true"
		}
		return ""
	}, "study-116"))
	ctx := context.Background()
	fresh := func(t *testing.T) *Sqlite {
		s, err := NewInMemory()
		require.NoError(t, err)
		t.Cleanup(func() { s.Close(ctx) })
		for _, q := range []string{
			`INSERT INTO listening_profiles(id,slug,display_name,created_at) VALUES(1,'study','Study',1),(2,'other','Other',1)`,
			`INSERT INTO artists(id) VALUES(1)`, `INSERT INTO artist_aliases VALUES(1,'Artista Ficticio','Study',1)`,
			`INSERT INTO releases(id) VALUES(1),(2),(3)`,
			`INSERT INTO release_aliases VALUES(1,'Album A','Study',1),(2,'Album B','Study',1),(3,'Album A (Deluxe)','Study',1)`,
			`INSERT INTO artist_releases VALUES(1,1,1),(1,2,1),(1,3,1)`,
		} {
			require.NoError(t, s.Exec(q))
		}
		return s
	}
	save := func(t *testing.T, s *Sqlite, album int32, title string, id uuid.UUID) int32 {
		tr, err := s.SaveTrack(ctx, db.SaveTrackOpts{AlbumID: album, Title: title, ArtistIDs: []int32{1}, RecordingMbzID: id, Duration: 180})
		require.NoError(t, err)
		return tr.ID
	}
	assoc := func(t *testing.T, s *Sqlite, album int32, title string, id uuid.UUID) int32 {
		tr, err := catalog.AssociateTrack(ctx, s, catalog.AssociateTrackOpts{AlbumID: album, TrackName: title, ArtistIDs: []int32{1}, TrackMbzID: id})
		require.NoError(t, err)
		return tr.ID
	}
	t.Run("34_217", func(t *testing.T) {
		s := fresh(t)
		mbid := uuid.MustParse("00000000-0000-0000-0000-000000001001")
		first := save(t, s, 1, "Faixa", mbid)
		second := assoc(t, s, 2, "Faixa", mbid)
		require.Equal(t, first, second)
		tr, err := s.GetTrack(ctx, db.GetTrackOpts{ID: second})
		require.NoError(t, err)
		require.EqualValues(t, 1, tr.AlbumID)
		a := assoc(t, s, 1, "Sem ID", uuid.Nil)
		b := assoc(t, s, 2, "Sem ID", uuid.Nil)
		require.NotEqual(t, a, b)
		require.NoError(t, s.MergeTracks(ctx, b, a))
		recreated := assoc(t, s, 2, "Sem ID", uuid.Nil)
		require.NotEqual(t, a, recreated)
		t.Logf("after merge, album B recreates track ID %d", recreated)
		t.Logf("same recording: IDs %d/%d, AlbumID=%d; no MBID: IDs %d/%d", first, second, tr.AlbumID, a, b)
	})
	t.Run("315", func(t *testing.T) {
		s := fresh(t)
		a := save(t, s, 1, "Intro", uuid.Nil)
		b := assoc(t, s, 1, "Intro", uuid.Nil)
		require.Equal(t, a, b)
		c := save(t, s, 1, "Intro", uuid.Nil)
		require.NotEqual(t, a, c)
		var n int
		require.NoError(t, s.db.QueryRow(`SELECT COUNT(*) FROM tracks_with_title WHERE title='Intro' AND release_id=1`).Scan(&n))
		require.Equal(t, 2, n)
		t.Logf("association coalesces %d/%d; direct save permits %d: %d candidates", a, b, c, n)
	})
	t.Run("315_distinct_MBID", func(t *testing.T) {
		s := fresh(t)
		one := uuid.MustParse("00000000-0000-0000-0000-000000001001")
		two := uuid.MustParse("00000000-0000-0000-0000-000000001002")
		a := save(t, s, 1, "Intro", one)
		tr, err := catalog.AssociateTrack(ctx, s, catalog.AssociateTrackOpts{AlbumID: 1, TrackName: "Intro", ArtistIDs: []int32{1}, TrackMbzID: two})
		require.NoError(t, err)
		require.Equal(t, a, tr.ID)
		var stored string
		require.NoError(t, s.db.QueryRow(`SELECT musicbrainz_id FROM tracks WHERE id=?`, a).Scan(&stored))
		require.Equal(t, one.String(), stored)
		require.Equal(t, two, *tr.MbzID)
		t.Logf("distinct MBIDs resolve to ID %d; stored=%s returned=%s", a, stored, tr.MbzID)
	})
	t.Run("369", func(t *testing.T) {
		s := fresh(t)
		a := save(t, s, 1, "Principal", uuid.Nil)
		require.NoError(t, s.Exec(`INSERT INTO track_aliases VALUES(1,'Alternativo','Study',0)`))
		b := assoc(t, s, 1, "Alternativo", uuid.Nil)
		require.NotEqual(t, a, b)
		t.Logf("secondary alias creates IDs %d/%d", a, b)
	})
	t.Run("359_360", func(t *testing.T) {
		s := fresh(t)
		a := save(t, s, 1, "Faixa", uuid.Nil)
		b := assoc(t, s, 1, "FAIXA", uuid.Nil)
		c := assoc(t, s, 1, "Faixa (Remaster)", uuid.Nil)
		d := assoc(t, s, 3, "Faixa", uuid.Nil)
		require.NotEqual(t, a, b)
		require.NotEqual(t, a, c)
		require.NotEqual(t, a, d)
		_, err := s.GetArtist(ctx, db.GetArtistOpts{Name: "ARTISTA FICTICIO"})
		require.ErrorIs(t, err, db.ErrNotFound)
		_, err = s.GetAlbum(ctx, db.GetAlbumOpts{Title: "ALBUM A", ArtistID: 1})
		require.ErrorIs(t, err, db.ErrNotFound)
		t.Logf("case/remaster/deluxe: IDs %d/%d/%d/%d; artist and album case miss", a, b, c, d)
	})
	t.Run("214_import", func(t *testing.T) {
		for _, withID := range []bool{false, true} {
			t.Run(fmt.Sprint(withID), func(t *testing.T) {
				s := fresh(t)
				mbid := ""
				if withID {
					mbid = "00000000-0000-0000-0000-000000001001"
					save(t, s, 1, "Homonima", uuid.MustParse(mbid))
				}
				json := fmt.Sprintf(`[{"track":[{"artist":{"#text":"Artista Ficticio"},"album":{"#text":"Album A"},"name":"Homonima","mbid":"%s","date":{"uts":"1000"}},{"artist":{"#text":"Artista Ficticio"},"album":{"#text":"Album B"},"name":"Homonima","mbid":"%s","date":{"uts":"1001"}}]}]`, mbid, mbid)
				file := filepath.Join(t.TempDir(), "recenttracks.json")
				require.NoError(t, os.WriteFile(file, []byte(json), 0600))
				result, err := importer.Upload(ctx, s, nil, 1, "lastfm", file)
				require.NoError(t, err)
				require.Equal(t, 2, result.Imported)
				want := 2
				if withID {
					want = 1
				}
				require.Equal(t, want, scalar(t, s, `SELECT COUNT(DISTINCT track_id) FROM listens WHERE profile_id=1`))
				require.Equal(t, want, scalar(t, s, `SELECT COUNT(DISTINCT t.release_id) FROM listens l JOIN tracks t ON t.id=l.track_id`))
				t.Logf("LastFM JSON: shared MBID=%v, %d tracks / %d albums / %d listens", withID, want, want, result.Imported)
			})
		}
	})
}
```
