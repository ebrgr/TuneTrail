# Ressalvas do Advisor após C3–C5

## Replay e erros de admissão

- **R1.** Se uma chave ativa já pertence a um recibo de sombra, a admissão devolve o recibo existente e preserva seu modo. Aquele recibo não entra na fila ativa. A janela ocorre quando um job fixado em sombra continua depois da purga e quando o mestre volta a ligar com o opt-in do perfil preservado e sombra acumulada. Na C6, reproduzir cada cenário, contar replays por modo do recibo e verificar quantos ficaram sem evento vinculado. Comparar com o histórico legado para distinguir escuta ausente de escuta já materializada.
- **R2.** O caminho `errReceiptKeyConflict` da admissão ativa sinaliza Revisar sem persistir uma decisão quando chaves aplicáveis diferentes do mesmo item apontam para recibos distintos. Os adaptadores atuais tornam esse caminho inalcançável. A API fornece a chave de cliente e o importador fornece a chave de arquivo. Um adaptador que combinar classes de chave precisa tratar o conflito de forma durável antes de confirmar o lote.
- **R3.** Com o mestre ligado e o perfil inativo, um BUSY em `BEGIN IMMEDIATE` também devolve 503 com `Retry-After`. O erro ocorre antes de a transação conseguir ler a flag do perfil. Com o mestre desligado, o escritor em sombra mantém o fail-open para essa classe de erro.
- **R4.** A descarga explícita de uma página de sombra devolve `ctx.Err()` se o job já foi cancelado. A descarga de encerramento em `defer` continua descartando esse erro. A falha de persistência da sombra continua apenas registrada em log.

## Janela de manutenção para o piloto

- **R5.** O lease atual é fixo em 120 s. Na C6, medir a duração da resolução e o tempo de retomada depois de interromper o primeiro worker. Registrar a distribuição de latência, a proporção de retomadas e rejeições de CAS.
- **R7.** Na C6, usar 300 mil recibos sintéticos no perfil a ativar. Medir separadamente a prévia e o tempo total sob `BEGIN IMMEDIATE`, incluindo recálculo do digest, purga por cascata e commit. Medir a espera dos outros escritores e os 503 durante a operação. Usar esses resultados para dimensionar a janela.
- Antes do piloto, reservar uma janela de manutenção. Suspender admissões que escrevam no SQLite e encerrar ou aguardar os jobs fixados em sombra. Exportar a prévia agregada do perfil, guardar o backup verificado e confirmar com o digest da prévia. Em caso de conflito, gerar nova prévia. Só retomar admissões depois de confirmar a transação de ativação e o estado do interruptor mestre.

## Ressalvas do Reviewer para a C6

- **R-a.** O resolvedor incrementa `attempt_count`, mas o contador ainda não limita a política de retry. Uma falha persistente pode voltar à fila indefinidamente. Na C6, injetar falhas transitórias e persistentes, medir tentativas e trabalho repetido, e decidir o teto e a política de retenção antes de introduzir uma fila de falhas definitivas.
- **R-b.** Com o mestre ligado, se a transação falha com um erro genérico antes da leitura da flag do perfil, a API retorna 500. O mesmo perfil inativo poderia seguir pelo escritor legado se a flag já fosse conhecida. Na C6, medir o comportamento por classe de erro e decidir a política de fallback para modo indeterminado.
