---
id: ISS-040
title: 040 — Persistência e concorrência da configuração
status: Done
assignee: []
created_date: '2026-11-12T23:39:00Z'
labels:
  - config
milestone: m-4
dependencies:
  - ISS-038
  - ISS-039
references: []
type: feature
ordinal: 40000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
> Importada de `docs-dev/issues/040-persistencia-configuracao-solved.md` — numeração local 040. Original removido em 12/11/2026; consulte o histórico do Git.

**Status:** Fechada e aprovada pelo usuário em 09/11/2026 · **Depende de:** 038 e 039

## Objetivo

Salvar os dez campos do [contrato da 038](<../docs/design/doc-019 - Issue-038-—-contrato-e-plano-da-configuração-administrativa.md>) em
`conf.toml`, com validação, revisão e preservação do documento anterior em falhas
anteriores à publicação. Nenhum campo listado recebe tratamento de segredo.

## Aceite

- Receber valores comuns, incluindo Last.fm e params Subsonic. Texto vazio limpa
  quando permitido pelo domínio; não criar protocolo especial manter/substituir/limpar.
  Não permitir escolha de caminhos `_FILE` pela interface.
- Validar arquivo+defaults e composição efetiva. Informar overrides e diferença
  entre salvo/ativo; todos os dez campos globais exigem reinício externo.
- Comparar revisão atual dentro do lock, detectar formulário/arquivo obsoleto e
  retornar conflito 409. Editor externo deve cooperar com lock ou parar serviço;
  não prometer exclusão universal de corridas externas.
- Preservar comentários e texto fora dos campos editados; rejeitar escrita quando
  não puder preservar ou quando o arquivo contiver chave/schema não suportados.
- Usar temporário no diretório de dados, permissões compatíveis, sync e publicação
  atômica; tratar links e limpar temporários. Antes do rename, erro mantém arquivo
  e revisão anteriores; depois, falha de sync exige reler e informar durabilidade
  incerta, sem declarar falsamente que nada foi alterado.
- O leitor de configuração deve permitir arquivo somente leitura; gravação
  indisponível deve ser informada. O servidor/SQLite ainda exigem dados graváveis. Não mudar env, Compose, diretório do banco ou relay SQLite.
- Testar duas abas, revisão obsoleta, edição externa cooperativa, comentários,
  readonly e falhas pré/pósrename com dados fictícios.

## Implementação e validação — 09/11/2026

Serviço interno `cfg.SettingsStore` implementado para os dez campos: revisão,
origem salva/efetiva, fotografia ativa, detecção de reinício, bloqueio cooperativo,
preservação de comentários, validação e publicação atômica. Retorna erros tipados
para a integração HTTP da 041. Não foi criada interface administrativa.

[Relatório completo, limitações e testes](../evidencias/configuracao/ISSUE_040.md).
A suíte Go, o detector de corridas do pacote, builds Linux e jornadas multiusuário
passaram. Advisor aprovou após correções e regressões dirigidas. A issue foi fechada por aceite do usuário.

## Encerramento — 09/11/2026

Fechada e aprovada por solicitação explícita do usuário. A interface administrativa segue na issue 041.
<!-- SECTION:DESCRIPTION:END -->
