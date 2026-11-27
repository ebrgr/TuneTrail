---
id: decision-012
title: Metadados descritivos por entidade com cadastro de fontes
date: '2026-11-27T21:34:00Z'
status: accepted
---
## Context

A ISSUE-218 mediu o custo de gravar os metadados descritivos em `metadata_values`, uma cópia por escuta: cerca de 3.807 bytes por escuta para oito valores, contra cerca de 2.400 bytes hoje por escuta nova inteira (`backlog/evidencias/issue-218/MEDICAO.md`). Gênero, ano, datas, ISRC e BPM descrevem a música, o lançamento ou o artista, e se repetem a cada reprodução.

## Decision

Escolha do usuário em 27/11/2026 (modelo D das alternativas medidas):

- Os metadados descritivos (gênero, tag, ano, datas, país de origem, ISRC e BPM) ficam em uma tabela nova, vinculados à gravação ou faixa, ao lançamento ou ao artista, por perfil, e não à escuta.
- A fonte e a instância ficam em um cadastro de fontes; cada valor aponta para ele. Origens diferentes continuam em linhas separadas.
- Os MBIDs continuam em `metadata_values`, por escuta, sem mudança.
- A migração única de metadados antes da ISSUE-195 (decision-011, A5) continua valendo e passa a criar essas tabelas.

## Consequences

- Não fica registrado qual escuta trouxe cada valor descritivo.
- Apagar escutas não remove o valor em cascata: ele sai quando o perfil deixa de ter escutas daquela entidade.
- Mesclagens de catálogo, desfazer mesclagem, exclusões e remoção de perfil precisam tratar a tabela nova.
- Substitui, nas ISSUE-217, 218, 219 e 055 e no doc-048, os trechos que previam esses tipos em `metadata_values`.
