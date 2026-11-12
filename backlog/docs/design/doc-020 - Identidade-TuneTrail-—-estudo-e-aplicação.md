---
id: doc-020
title: Identidade TuneTrail — estudo e aplicação
type: specification
created_date: '2026-11-12T23:40:00Z'
updated_date: '2026-11-12T23:40:00Z'
tags:
  - design
---
> Importado de [docs-dev/IDENTIDADE_TUNETRAIL.md](../../../docs-dev/IDENTIDADE_TUNETRAIL.md).

# Identidade TuneTrail — estudo e aplicação

Em 07/11/2026 o item 2 do roadmap foi colocado em espera e o item 4 passou a ser priorizado. A issue local [043](../../../docs-dev/issues/043-identidade-tunetrail-solved.md) registra o trabalho.

O SVG e o novo PNG foram aceitos pelo usuário em 07/11/2026. O PNG atualizado foi usado no README, na documentação e nos ícones derivados do navegador e de instalação. O ícone original do Koito é preservado para crédito. A base original documentada é Koito v0.3.2; a versão do fork continua separada.

Foram aplicados TuneTrail, o slogan e o crédito no README e no rodapé público; títulos, seleção de perfil, administração, manifesto, favicon e ícones de instalação receberam a nova identidade. A página inicial da documentação e sua navegação também usam o novo nome. Os links de demonstração e doação foram retirados. Documentação histórica e referências técnicas ao Koito original permanecem para compatibilidade e atribuição.

O Advisor aprovou a direção visual com ressalvas quanto a substituições globais: `KOITO_*`, `koito.db`, caminhos de dados, cookies, formatos de importação, módulo Go, rotas e imagem existente foram preservados. Não foi inventado domínio nem renomeado o repositório remoto. Uma futura migração desses contratos exige planejamento próprio.

Validação: `make version.check`, análise XML dos SVGs, manifesto JSON e testes `go test ./internal/webui ./internal/adminui` passaram (cache Go temporário). A tentativa de iniciar a prévia falhou porque o ambiente bloqueou abertura de socket local; portanto não foram obtidas capturas nem realizada comparação visual no navegador. O usuário aceitou os dois arquivos de identidade; a comparação da interface em desktop e celular e a compilação da documentação permanecem sem verificação neste ambiente.

## Encerramento e continuidade — 09/11/2026

O usuário aprovou o fechamento das issues 043–045. Os resultados e limitações
anteriores são preservados como histórico, sem afirmar novas verificações.
A documentação e o site, incluindo endereço próprio, identidade e capturas,
passam à [issue 048](../../../docs-dev/issues/048-documentacao-site-tunetrail.md).
