# Candidatas a restringir em uma issue futura

As nove candidatas são combinações de tipo e ação aceitas por `IsAction` que sempre falham no handler ou serviço para aquele tipo. A proposta é fechar essas combinações no dispatcher e manter os handlers compartilhados. Nenhuma rota, operação ou permissão foi alterada nesta auditoria.

A busca cruzou `assets/templates/admin/manage.html`, `assets/js/media-dialog.js`, a matriz `test_assets/admin-web-ui-routes.json`, os percursos Python e CJS em `scripts/` e os testes de catálogo e de escutas. O E2E de permissões chama as nove variantes como sondas negativas e recebe 401/403 antes de executar a operação. Esses acessos precisam de atualização explícita se a recusa passar a 404. Antes de qualquer mudança, fixe o comportamento atual em teste com sessão, perfil e formulário válidos e verifique que o estado do SQLite permanece igual.

| Candidata POST | Chamadores verificados e restrição atual | Risco | Teste necessário |
|---|---|---|---|
| `/{slug}/manage/album/{id}/artist-add` | O formulário de crédito em `manage.html` usa `/manage/track/`. `mediaAction` exige track. | Baixo. Muda a recusa do formulário fabricado. | Álbum recusado e adição de crédito válida na faixa. |
| `/{slug}/manage/album/{id}/artist-remove` | O formulário de remoção em `manage.html` usa track. O crédito principal de álbum usa `primary-artist`. | Baixo. Preservar a edição do crédito principal do álbum. | Remoção em álbum recusada, remoção de crédito secundário na faixa e edição principal do álbum preservadas. |
| `/{slug}/manage/album/{id}/listen-add` | `manage.html` monta a ação dentro do editor de faixa. `mediaAction` exige track. | Baixo. A UI de inclusão não usa esta variante. | Álbum recusado, inclusão em faixa com horário e fuso aceitos. |
| `/{slug}/manage/album/{id}/listen-delete` | O formulário de ocorrência aponta para faixa. Álbum usa revisão e `confirm` para exclusão pessoal em lote. | Baixo para a ação isolada. A exclusão pessoal em lote precisa permanecer disponível. | Exclusão direta por álbum recusada, exclusão exata por faixa e confirmação pessoal por álbum preservadas. |
| `/{slug}/manage/artist/{id}/artist-add` | Busca e inclusão de créditos em `manage.html` usam track. `mediaAction` exige track. | Baixo. Pode mudar o status observado por uma automação que testa a recusa. | Artista recusado, inclusão com ID de artista válida na faixa. |
| `/{slug}/manage/artist/{id}/artist-remove` | O formulário usa track e o handler rejeita outros tipos. | Baixo. Preservar regras de remoção de crédito e invariante de crédito principal da faixa. | Artista recusado, faixa aceita apenas para crédito removível. |
| `/{slug}/manage/artist/{id}/listen-add` | Formulário e fluxo em `validate_listen_dialogs.cjs` usam faixa. `mediaAction` exige track. | Baixo. Muda somente a recusa da combinação inválida. | Artista recusado, inclusão por faixa continua vinculada ao dono. |
| `/{slug}/manage/artist/{id}/listen-delete` | O formulário de escuta usa faixa. Exclusão pessoal de artista usa revisão e `confirm`. | Baixo para a ação isolada. Há risco de confundir a ação direta com a confirmação de lote. | Exclusão direta por artista recusada, preservando exclusão exata por faixa e lote pessoal por artista. |
| `/{slug}/manage/artist/{id}/primary-artist` | O formulário genérico de créditos rende opções em álbum e faixa. `SetMediaArtistPrimary` só implementa esses dois tipos. | Baixo. O formulário compartilhado precisa continuar funcionando nos tipos válidos. | Artista recusado, alteração de crédito principal em álbum e faixa preservada. |

## Operações que dependem de decisão

A ISSUE-149 é a referência para avaliar a exclusão global em `DELETE /apis/web/v1/artist/{id}`, `DELETE /apis/web/v1/album/{id}` e `DELETE /apis/web/v1/track/{id}`. As exclusões pessoais da UI seguem outro contrato. `TestDelete` ainda cobre a API global, então há comportamento explícito a preservar até resolver o destino da API e identificar consumidores. `DELETE /apis/web/v1/data` também requer decisão própria sobre manutenção e automação root.

Os aliases sem slug, redirects de páginas antigas, URLs legadas de imagem, exportações, now-playing, health e compatibilidade ListenBrainz ficam para análise de contrato. O ensaio local não mede uso em instalações de terceiros.
