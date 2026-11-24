# Parecer da ISSUE-175

APROVADO

Revisados `030ebb1` e a correção `d49dd41`. O fallback conserva as posições vazias dos créditos, mantém a precedência dos IDs declarados na escuta e compara reenvios pelas projeções dos digests anteriores. Os testes dirigidos de ListenBrainz e enriquecimento passaram, incluindo o cenário com crédito parcial a partir do JSON. A revisão confirmou que a branch não acrescenta migração.

O lookup usa o relay aprovado para a API oficial do ListenBrainz no perfil da escuta. O provedor retorna um candidato canônico. Correspondências de texto iguais entre versões indistinguíveis continuam limitadas à evidência disponível e os resultados permanecem declarações por perfil.
