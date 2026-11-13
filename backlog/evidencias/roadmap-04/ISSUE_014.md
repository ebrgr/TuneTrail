# Issue 014 — revisão de exclusões e mesclagens

**Atualização na issue 015:** integração Go, jornadas reais e comparação visual
registradas em [ISSUE_015.md](ISSUE_015.md). As notas abaixo descrevem a entrega
anterior à integração; o aceite final do usuário permanece pendente.

Núcleo implementado em 05/11/2026. [Issue](<../../tasks/issue-014 - 014-—-Migrar-confirmações-de-deleção-e-mesclagem-para-Go.md>) · [Plano](<../../docs/roadmaps/doc-007 - Roadmap-4-—-administração-importação-chaves-edição-deleção-e-mesclagem.md>).

O handler Go isolado prepara uma revisão seguida de confirmação por POST para
artistas, álbuns e faixas. Mostra origem, destino e ID sobrevivente, aliases,
MBIDs, artistas, capas, relações, escutas e perfis da origem e do destino. O
cálculo de exclusão de artista inclui a cascata de álbuns cujo último artista de
álbum seria removido, mesmo quando outras pessoas estão associadas às faixas.
Escutas em faixas dos álbuns relacionados também entram no impacto de artista.

A revisão é vinculada à sessão, perfil e rota, tem prazo de dez minutos e uso
único; cada POST exige sessão, origem e CSRF válidos. A confirmação recalcula o
fingerprint e executa a operação no mesmo tx. Conteúdo desatualizado, origem ou
destino removido e conflitos de escrita WAL pedem nova revisão com 409. A
revisão conservadora inclui **todo** o catálogo e histórico: mudanças não
relacionadas também podem invalidá-la, e o custo cresce com o banco. Não há
persistência dos planos ao reiniciar nem suporte distribuído entre processos.

A limpeza global deixou de rodar em merges e na exclusão de faixa: catálogo
sem escutas fora da operação e o destino vazio são preservados. As regras de
merge permanecem explícitas: destino conserva MBID, aliases e metadados;
aliases da origem são removidos. Relações de artista são transferidas; álbum
move faixas e associa artistas de origem como secundários; merge de faixa move
escutas e não transfere suas relações, aliases ou duração. Capa da origem só
substitui a do destino quando solicitado e existente. Duplicatas de faixa são
consolidadas apenas por perfil e instante; perfis distintos são preservados.

A exclusão de perfil em Go confirma nome e contagem no DELETE atômico; o último
perfil pode ser removido sem remover o administrador ou o catálogo. A API JSON
mantém seus payloads e exclusões idempotentes, compartilhando as correções SQLite.
Após confirmação Go, caches de reprodução afetados são invalidados; imagens de
catálogo órfãs aguardam o coletor global, sem prometer rollback após commit.

## Verificação

- `go test ./...` e testes de corrida dos pacotes administrativos e SQLite.
- SQLite v3 descartável: auto-merge, IDs inválidos, inexistência, rollback por
  trigger, origem/destino sem escutas, catálogo não relacionado, cascata real de
  artista, perfis comuns e deduplicação em ambas as direções.
- Teste WAL com duas conexões reproduz snapshot antigo e confirma conflito sem
  perda de escutas; alteração de alias com contagens iguais invalida revisão.
- HTTP isolado: sessão, CSRF, rota e perfil vinculados, confirmação de uso único,
  plano obsoleto ou expirado, direção inversa, dados pessoais separados, cache
  de reprodução e último perfil.
- Validações de integração em bases novas/migradas v3 e servidores locais.
- Advisor (Sol) revisou a transação, cascatas e impacto; parecer final aprovado.

O handler segue isolado até a 015. Busca assistida de destino, jornadas completas
na interface integrada, verificações de carga do fingerprint e aceite visual
continuam pendentes; capturas desktop/celular serão feitas somente na 015.
