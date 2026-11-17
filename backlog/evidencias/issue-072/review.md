APROVADO

[OK] internal/webui/home.go — totais históricos independentes do período dos KPIs, com estados de erro separados.
[OK] internal/webui/home_kpis.go — variação zero agora mostra “Igual”; percentuais sem sinal com “Mais/Menos” e “Novo” preservam as situações distintas.
[OK] assets/css/ui/pages/home.css — totais à esquerda e atividade à direita em >=64rem; empilhamento e colunas por container abaixo disso.
[OK] assets/templates/public/graphs.html — data-tip traduzido, sem title nativo; tabela acessível preservada.
[OK] assets/css/ui/components/activity.css — hover visível e tooltips dentro do cartão em 320/390/768/1440px; regra de foco presente, sem JavaScript.
[OK] Makefile — referência final da ISSUE-072 validada por ui.check; make check e ui.check verdes.

[OK] assets/css/ui/pages/home.css — comparação pequena ao lado, baseline e espaço de 12px, com quebra flexível; sem overflow em 390/1440px.
[OK] internal/webui/i18n_051_test.go — textos en/pt-BR verificados; sem comparação em all_time e sem duplicação por aria-label.
