APROVADO

[OK] internal/webui/home_kpis.go:44 — intervalos adjacentes sem sobreposição, base zero explícita e ausência de comparação histórica; testes SQLite verdes.
[OK] internal/db/sqlite/listen_activity.go:286 — consulta de sequência limitada ao intervalo e ao perfil; caminhos legados preservados.
[OK] assets/templates/public/live.html:139 — resumo traduzido, tocando agora no topo e grade existente preservada.
[OK] assets/css/ui/pages/home.css:160 — tokens e largura fluida; matriz de navegador sem overflow, seletor sticky verificado.
[OK] Makefile:159 — referência da ISSUE-072 comparada por ui.check; make check e ui.check verdes.
[RISCO] assets/css/ui/components/activity.css:35 — células de aproximadamente 6,6px em 320px servem para densidade; leitura individual depende de aceite. Sem mudança no heatmap e sem rolagem; AC2/AC4 pendentes.
