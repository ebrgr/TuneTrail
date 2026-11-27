# ISSUE-218 — instruções para retomar (modelo D, decision-012)

O usuário aprovou o modelo D em 27/11/2026. Continuam valendo o prompt original e os limites dele; o que muda está abaixo.

## Modelo
- Migração 026, a única de metadados antes da ISSUE-195: cria um cadastro de fontes (fonte e instância, sem repetição) e a tabela de metadados descritivos por perfil e entidade, com os tipos da 218 e da 217 (gênero, tag, ano, datas, país de origem, ISRC, BPM, primeiro lançamento do grupo e primeira publicação da gravação). Sem `event_id`.
- `metadata_values` e os MBIDs por escuta **não mudam**; não recrie a tabela nem os gatilhos dela. Os tipos não-mbid do CHECK antigo ficam sem uso.
- Atenção ao vínculo da faixa: mesclar gravações, e desfazer a mesclagem, move `occurrences.recording_id`. Um valor ligado só à gravação não pode ser separado de volta. Avalie ligar os valores da faixa à ocorrência, que é estável; escolha, justifique no HANDOFF e mantenha o tamanho próximo ao medido para o modelo D.

## Regras
- Isolamento por perfil também no banco: só grava se o perfil tiver escuta que sustente a entidade.
- Ciclo de vida, com teste para cada caso: excluir escuta, expurgo e descarte; excluir faixa, álbum e artista; remover perfil; mesclar e desfazer mesclagem. O valor sai quando o perfil deixa de ter escutas da entidade e acompanha a mesclagem sem duplicar.
- Campo ausente ou vazio não apaga dado anterior; o mesmo valor da mesma fonte não se repete. Documente o que `observed_at` significa.
- A instância da fonte não é exportada nem registrada em log.
- O formato de exportação não muda. Se for necessário mudar, pare e pergunte.

## Medição
Meça de novo com a migração final (1, 3 e 10 escutas por faixa; incluindo o cadastro de fontes) e atualize `MEDICAO.md`. Referência: 573, 189 e 56 bytes por escuta. O limite de 500 bytes do prompt original foi substituído: pare e pergunte se o pior caso passar de cerca de 650 bytes.
