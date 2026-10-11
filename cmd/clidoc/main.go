package main

import (
	"flag"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/lukasalmeida/clidoc/internal/config"
	"github.com/lukasalmeida/clidoc/internal/generator"
	"github.com/lukasalmeida/clidoc/internal/ir"
	"github.com/lukasalmeida/clidoc/internal/parser"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "init":
		cmdInit(os.Args[2:])
	case "generate":
		cmdGenerate(os.Args[2:])
	case "serve":
		cmdServe(os.Args[2:])
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Uso: clidoc <init|generate|serve> [opções]")
}

func cmdInit(args []string) {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	configPath := fs.String("config", "clidoc.yaml", "caminho do arquivo de configuração")
	fs.Parse(args)

	if err := config.WriteDefault(*configPath); err != nil {
		fmt.Fprintf(os.Stderr, "erro ao criar config: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Arquivo de configuração criado em %s\n", *configPath)
}

func cmdGenerate(args []string) {
	fs := flag.NewFlagSet("generate", flag.ExitOnError)
	configPath := fs.String("config", "clidoc.yaml", "arquivo de configuração")
	source := fs.String("source", "", "diretório para analisar")
	output := fs.String("output", "", "arquivo de saída")
	format := fs.String("format", "", "formato de saída: openapi ou markdown")
	title := fs.String("title", "", "título da API")
	version := fs.String("version", "", "versão da API")
	baseURL := fs.String("baseURL", "", "base URL da API")
	fs.Parse(args)

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "erro ao ler configuração: %v\n", err)
		os.Exit(1)
	}
	if *source == "" {
		*source = cfg.Source
	}
	if *output == "" {
		*output = cfg.Output
	}
	if *title == "" {
		*title = cfg.Title
	}
	if *version == "" {
		*version = cfg.Version
	}
	if *baseURL == "" {
		*baseURL = cfg.BaseURL
	}
	if *format == "" {
		*format = detectFormat(*output)
	}
	if *format == "" {
		*format = "openapi"
	}

	routes, err := parser.ParseDirectory(*source)
	if err != nil {
		fmt.Fprintf(os.Stderr, "erro ao analisar código: %v\n", err)
		os.Exit(1)
	}

	api := ir.API{
		Title:     *title,
		Version:   *version,
		BaseURL:   *baseURL,
		Endpoints: routes,
	}

	switch strings.ToLower(*format) {
	case "markdown", "md":
		if err := generator.WriteMarkdown(*output, api); err != nil {
			fmt.Fprintf(os.Stderr, "erro ao escrever Markdown: %v\n", err)
			os.Exit(1)
		}
	case "openapi", "json":
		if err := generator.WriteFile(*output, api); err != nil {
			fmt.Fprintf(os.Stderr, "erro ao escrever OpenAPI: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "formato inválido: %s (use openapi ou markdown)\n", *format)
		os.Exit(1)
	}

	fmt.Printf("Gerado %s em formato %s com %d rotas\n", *output, *format, len(routes))
}

func detectFormat(path string) string {
	if strings.HasSuffix(strings.ToLower(path), ".md") {
		return "markdown"
	}
	if strings.HasSuffix(strings.ToLower(path), ".json") || strings.HasSuffix(strings.ToLower(path), ".yaml") || strings.HasSuffix(strings.ToLower(path), ".yml") {
		return "openapi"
	}
	return ""
}

func cmdServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	configPath := fs.String("config", "clidoc.yaml", "arquivo de configuração")
	port := fs.Int("port", 8080, "porta do servidor")
	fs.Parse(args)

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "erro ao ler configuração: %v\n", err)
		os.Exit(1)
	}
	if *port != 8080 {
		cfg.Port = *port
	}

	if _, err := os.Stat(cfg.Output); err != nil {
		if os.IsNotExist(err) {
			generateForServe(cfg)
		} else {
			fmt.Fprintf(os.Stderr, "erro ao verificar arquivo de saída: %v\n", err)
			os.Exit(1)
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/openapi.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		data, err := os.ReadFile(cfg.Output)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_, _ = w.Write(data)
	})
	mux.HandleFunc("/icon.png", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "doc/images/icon.png")
	})
	mux.HandleFunc("/icon_transp.png", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "doc/images/icon_transp.png")
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		_ = template.Must(template.New("index").Parse(pageTemplate)).Execute(w, nil)
	})

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	fmt.Printf("Servidor de pré-visualização rodando em http://%s\n", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		fmt.Fprintf(os.Stderr, "erro ao iniciar servidor: %v\n", err)
		os.Exit(1)
	}
}

func generateForServe(cfg config.Config) {
	routes, err := parser.ParseDirectory(cfg.Source)
	if err != nil {
		fmt.Fprintf(os.Stderr, "erro ao analisar código: %v\n", err)
		os.Exit(1)
	}
	api := ir.API{Title: cfg.Title, Version: cfg.Version, BaseURL: cfg.BaseURL, Endpoints: routes}
	if err := generator.WriteFile(cfg.Output, api); err != nil {
		fmt.Fprintf(os.Stderr, "erro ao gerar arquivo de documentação: %v\n", err)
		os.Exit(1)
	}
}

const pageTemplate = `<!doctype html>
<html lang="pt-BR">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <meta name="theme-color" content="#0b0f17" />
    <title>CliDoc | Documentação da API</title>
    <link rel="icon" type="image/png" href="/icon.png" />
    <link rel="preconnect" href="https://fonts.googleapis.com" />
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin />
    <link href="https://fonts.googleapis.com/css2?family=Geist:wght@400;500;600;700&family=JetBrains+Mono:wght@400;500;600;700&display=swap" rel="stylesheet" />
    <script>
      try {
        document.documentElement.dataset.theme =
          localStorage.getItem('clidoc-theme') === 'light' ? 'light' : 'dark';
      } catch (_) {}
    </script>
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.11.0/swagger-ui.css" />
    <style>
      :root {
        color-scheme: dark;
        font-family: Geist, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
        background: #0b0f17;
        color: #e2e8f0;
        --page-bg: #0b0f17;
        --surface-bg: #111827;
        --surface-muted: #182234;
        --surface-code: #0d131f;
        --text-primary: #e2e8f0;
        --text-muted: #94a3b8;
        --border-color: rgb(255 255 255 / 8%);
        --accent: #ff6b00;
        --accent-hover: #ff7a1a;
        --focus-border: rgb(255 107 0 / 50%);
        --shadow: 0 4px 20px -2px rgb(0 0 0 / 60%);
      }
      :root[data-theme="light"] {
        color-scheme: light;
        background: #f3f6fa;
        color: #1e293b;
        --page-bg: #f3f6fa;
        --surface-bg: #fff;
        --surface-muted: #f8fafc;
        --surface-code: #eef2f7;
        --text-primary: #1e293b;
        --text-muted: #526174;
        --border-color: #dce3ec;
        --accent: #e85d04;
        --accent-hover: #c94f00;
        --focus-border: rgb(232 93 4 / 45%);
        --shadow: 0 4px 18px rgb(15 23 42 / 7%);
      }
      * { box-sizing: border-box; }
      body { margin: 0; background: var(--page-bg); color: var(--text-primary); }
      .site-header {
        position: sticky;
        top: 0;
        z-index: 10;
        background: rgb(17 24 39 / 85%);
        border-bottom: 1px solid rgb(255 255 255 / 8%);
        backdrop-filter: blur(16px);
        color: #fff;
      }
      .site-header__content {
        max-width: 1440px;
        min-height: 72px;
        margin: 0 auto;
        padding: 14px 32px;
        display: flex;
        align-items: center;
        gap: 14px;
      }
      .site-header__brand { display: flex; align-items: center; gap: 14px; }
      .theme-toggle {
        margin-left: auto;
        padding: 9px 12px;
        border: 1px solid rgb(255 255 255 / 12%);
        border-radius: 6px;
        background: transparent;
        color: #fff;
        font: inherit;
        font-size: 13px;
        transition: border-color 150ms ease, background 150ms ease, color 150ms ease;
        cursor: pointer;
      }
      .theme-toggle:hover { border-color: var(--focus-border); background: rgb(255 255 255 / 5%); color: #ff9f43; }
      .theme-toggle:focus-visible { outline: 2px solid #ff6b00; outline-offset: 3px; }
      .brand-icon { width: 42px; height: 42px; border-radius: 8px; object-fit: cover; }
      .brand-name { margin: 0; font-size: 18px; font-weight: 600; letter-spacing: -.03em; }
      .brand-caption { margin: 3px 0 0; color: #94a3b8; font-size: 12px; }
      #swagger-ui { max-width: 1440px; margin: 0 auto; padding: 32px; }
      .swagger-ui { color: var(--text-primary); }
      .swagger-ui .topbar { display: none; }
      .swagger-ui .info { margin: 4px 0 32px; }
      .swagger-ui .info .title { color: var(--text-primary); font-size: 30px; font-weight: 600; letter-spacing: -.025em; }
      .swagger-ui .info p,
      .swagger-ui .info li,
      .swagger-ui .markdown p,
      .swagger-ui .markdown li { color: var(--text-muted); }
      .swagger-ui .info a,
      .swagger-ui .markdown a { color: var(--accent-hover); }
      .swagger-ui .scheme-container {
        padding: 18px 20px;
        border: 1px solid var(--border-color);
        border-radius: 8px;
        background: var(--surface-bg);
        backdrop-filter: blur(12px);
        box-shadow: var(--shadow);
      }
      .swagger-ui .opblock-tag {
        color: var(--text-primary);
        border-bottom-color: var(--border-color);
        font-size: 18px;
      }
      .swagger-ui .opblock {
        border-radius: 8px;
        box-shadow: var(--shadow);
      }
      .swagger-ui .opblock .opblock-summary { border-radius: 8px; }
      .swagger-ui .btn.authorize {
        border-color: var(--accent);
        color: var(--accent-hover);
      }
      .swagger-ui .btn.authorize svg { fill: var(--accent-hover); }
      .swagger-ui .btn.execute {
        border-color: var(--accent);
        background: var(--accent);
        color: #0b0f17;
        font-weight: 700;
        box-shadow: inset 0 1px rgb(255 255 255 / 20%);
      }
      .swagger-ui .btn.execute:hover {
        border-color: var(--accent-hover);
        background: var(--accent-hover);
        box-shadow: 0 0 16px -2px rgb(255 107 0 / 35%);
      }
      .swagger-ui section.models {
        border-color: var(--border-color);
        border-radius: 9px;
      }
      .swagger-ui .btn:not(.authorize):not(.execute) { border-color: var(--border-color); color: var(--text-primary); }
      .swagger-ui .btn:not(.authorize):not(.execute):hover { border-color: var(--accent); color: var(--accent-hover); }
      .swagger-ui .model-box,
      .swagger-ui .model-container { background: var(--surface-muted); }
      .swagger-ui .model-title,
      .swagger-ui .parameter__name,
      .swagger-ui .response-col_status { color: var(--text-primary); }
      .swagger-ui table thead tr td,
      .swagger-ui table thead tr th,
      .swagger-ui .responses-inner h4,
      .swagger-ui .responses-inner h5 { color: var(--text-primary); }
      .swagger-ui .opblock-description-wrapper,
      .swagger-ui .opblock-section-header,
      .swagger-ui .responses-wrapper,
      .swagger-ui .responses-inner,
      .swagger-ui .model-box,
      .swagger-ui .model-container,
      .swagger-ui table thead tr td,
      .swagger-ui table thead tr th { background: var(--surface-muted); }
      .swagger-ui .opblock-description-wrapper p,
      .swagger-ui .opblock-external-docs-wrapper p,
      .swagger-ui .response-col_description,
      .swagger-ui .parameter__type,
      .swagger-ui .parameter__in { color: var(--text-muted); }
      :root[data-theme="dark"] .swagger-ui,
      :root[data-theme="dark"] .swagger-ui .opblock-body,
      :root[data-theme="dark"] .swagger-ui .opblock-section-header,
      :root[data-theme="dark"] .swagger-ui .responses-wrapper,
      :root[data-theme="dark"] .swagger-ui .responses-inner,
      :root[data-theme="dark"] .swagger-ui .response,
      :root[data-theme="dark"] .swagger-ui .model-box,
      :root[data-theme="dark"] .swagger-ui .model-container,
      :root[data-theme="dark"] .swagger-ui .model,
      :root[data-theme="dark"] .swagger-ui table thead tr td,
      :root[data-theme="dark"] .swagger-ui table thead tr th,
      :root[data-theme="dark"] .swagger-ui .dialog-ux .modal-ux { background: var(--surface-bg); }
      :root[data-theme="dark"] .swagger-ui .opblock-description-wrapper,
      :root[data-theme="dark"] .swagger-ui .opblock-external-docs-wrapper,
      :root[data-theme="dark"] .swagger-ui .tab li,
      :root[data-theme="dark"] .swagger-ui .parameter__extension,
      :root[data-theme="dark"] .swagger-ui .highlight-code,
      :root[data-theme="dark"] .swagger-ui .microlight { background: var(--surface-muted); }
      :root[data-theme="dark"] .swagger-ui .opblock-tag,
      :root[data-theme="dark"] .swagger-ui .opblock-tag small,
      :root[data-theme="dark"] .swagger-ui .opblock-summary-description,
      :root[data-theme="dark"] .swagger-ui .opblock-description-wrapper p,
      :root[data-theme="dark"] .swagger-ui .opblock-external-docs-wrapper p,
      :root[data-theme="dark"] .swagger-ui .opblock-section-header h4,
      :root[data-theme="dark"] .swagger-ui .opblock-section-header label,
      :root[data-theme="dark"] .swagger-ui .responses-inner h4,
      :root[data-theme="dark"] .swagger-ui .responses-inner h5,
      :root[data-theme="dark"] .swagger-ui .response-col_description,
      :root[data-theme="dark"] .swagger-ui .parameter__name,
      :root[data-theme="dark"] .swagger-ui .parameter__type,
      :root[data-theme="dark"] .swagger-ui .parameter__in,
      :root[data-theme="dark"] .swagger-ui .model-title,
      :root[data-theme="dark"] .swagger-ui .model,
      :root[data-theme="dark"] .swagger-ui table thead tr td,
      :root[data-theme="dark"] .swagger-ui table thead tr th { color: var(--text-primary); }
      :root[data-theme="dark"] .swagger-ui input[type="text"],
      :root[data-theme="dark"] .swagger-ui input[type="password"],
      :root[data-theme="dark"] .swagger-ui input[type="email"],
      :root[data-theme="dark"] .swagger-ui textarea,
      :root[data-theme="dark"] .swagger-ui select {
        border-color: var(--border-color);
        background: var(--surface-bg);
        color: var(--text-primary);
      }
      :root[data-theme="dark"] .swagger-ui .responses-table td,
      :root[data-theme="dark"] .swagger-ui .responses-table th,
      :root[data-theme="dark"] .swagger-ui .model,
      :root[data-theme="dark"] .swagger-ui .model-box,
      :root[data-theme="dark"] .swagger-ui .model-container { border-color: var(--border-color); }
      :root[data-theme="dark"] .swagger-ui .prop-type { color: #fdba74; }
      .swagger-ui,
      .swagger-ui .wrapper { color: var(--text-primary); }
      .swagger-ui .opblock-tag {
        border-bottom-color: var(--border-color);
        font-weight: 600;
        letter-spacing: -.01em;
      }
      .swagger-ui .opblock-tag small,
      .swagger-ui .opblock-summary-description { color: var(--text-muted); }
      .swagger-ui .opblock .opblock-summary-method,
      .swagger-ui .opblock .opblock-summary-path,
      .swagger-ui .opblock .opblock-summary-path__deprecated,
      .swagger-ui .parameter__name,
      .swagger-ui .prop-name,
      .swagger-ui .model-title,
      .swagger-ui .response-col_status,
      .swagger-ui .microlight,
      .swagger-ui pre,
      .swagger-ui code,
      .swagger-ui .model,
      .swagger-ui .model-box { font-family: "JetBrains Mono", ui-monospace, SFMono-Regular, Menlo, monospace; }
      .swagger-ui .opblock .opblock-summary-path,
      .swagger-ui .opblock .opblock-summary-path__deprecated { font-size: 13px; }
      .swagger-ui .opblock .opblock-summary-method {
        min-width: 76px;
        border-radius: 4px;
        font-size: 11px;
        font-weight: 700;
        letter-spacing: .06em;
      }
      .swagger-ui .opblock.opblock-get { border-color: rgb(16 185 129 / 30%); background: rgb(16 185 129 / 6%); }
      .swagger-ui .opblock.opblock-get .opblock-summary-method { border-color: rgb(16 185 129 / 30%); background: rgb(16 185 129 / 12%); color: #34d399; }
      .swagger-ui .opblock.opblock-post { border-color: rgb(255 107 0 / 30%); background: rgb(255 107 0 / 6%); }
      .swagger-ui .opblock.opblock-post .opblock-summary-method { border-color: rgb(255 107 0 / 30%); background: rgb(255 107 0 / 12%); color: #ff8a33; }
      .swagger-ui .opblock.opblock-put,
      .swagger-ui .opblock.opblock-patch { border-color: rgb(56 189 248 / 30%); background: rgb(56 189 248 / 6%); }
      .swagger-ui .opblock.opblock-put .opblock-summary-method,
      .swagger-ui .opblock.opblock-patch .opblock-summary-method { border-color: rgb(56 189 248 / 30%); background: rgb(56 189 248 / 12%); color: #38bdf8; }
      .swagger-ui .opblock.opblock-delete { border-color: rgb(244 63 94 / 30%); background: rgb(244 63 94 / 6%); }
      .swagger-ui .opblock.opblock-delete .opblock-summary-method { border-color: rgb(244 63 94 / 30%); background: rgb(244 63 94 / 12%); color: #fb7185; }
      .swagger-ui .opblock .opblock-summary,
      .swagger-ui .opblock .opblock-body,
      .swagger-ui .opblock .opblock-section-header,
      .swagger-ui .opblock .responses-wrapper,
      .swagger-ui .opblock .responses-inner { background: transparent; }
      .swagger-ui .opblock .opblock-section-header,
      .swagger-ui .responses-inner h4,
      .swagger-ui .responses-inner h5 { border-bottom-color: var(--border-color); }
      .swagger-ui .opblock-body,
      .swagger-ui .opblock-section-header,
      .swagger-ui .responses-wrapper,
      .swagger-ui .responses-inner,
      .swagger-ui .response,
      .swagger-ui .model-box,
      .swagger-ui .model-container,
      .swagger-ui .model,
      .swagger-ui .dialog-ux .modal-ux { color: var(--text-primary); }
      .swagger-ui .opblock-description-wrapper,
      .swagger-ui .opblock-external-docs-wrapper,
      .swagger-ui .highlight-code,
      .swagger-ui .microlight,
      .swagger-ui .model-box,
      .swagger-ui .model-container { background: var(--surface-muted); }
      .swagger-ui .highlight-code,
      .swagger-ui .microlight { background: var(--surface-code); }
      .swagger-ui .highlight-code,
      .swagger-ui .microlight,
      .swagger-ui .curl-command { border: 1px solid var(--border-color); border-radius: 6px; }
      .swagger-ui .microlight { color: #e2e8f0; }
      .swagger-ui .microlight .string,
      .swagger-ui .microlight .value { color: #34d399; }
      .swagger-ui .microlight .number { color: #ff9f43; }
      .swagger-ui .microlight .boolean,
      .swagger-ui .microlight .keyword { color: #38bdf8; }
      .swagger-ui .microlight .comment { color: #64748b; }
      .swagger-ui .response-col_description__inner p,
      .swagger-ui .parameter__deprecated,
      .swagger-ui .response-col_links,
      .swagger-ui .markdown p,
      .swagger-ui .markdown li { color: var(--text-muted); }
      .swagger-ui .parameter__name.required span,
      .swagger-ui .required { color: #fb7185; }
      .swagger-ui .parameter__type,
      .swagger-ui .prop-type { color: #38bdf8; }
      .swagger-ui input[type="text"],
      .swagger-ui input[type="password"],
      .swagger-ui input[type="email"],
      .swagger-ui textarea,
      .swagger-ui select {
        border: 1px solid rgb(255 255 255 / 10%);
        border-radius: 6px;
        background: var(--surface-bg);
        color: var(--text-primary);
        font-family: "JetBrains Mono", ui-monospace, SFMono-Regular, Menlo, monospace;
      }
      .swagger-ui input:focus,
      .swagger-ui textarea:focus,
      .swagger-ui select:focus {
        border-color: var(--accent);
        outline: none;
        box-shadow: 0 0 0 1px var(--accent), 0 0 16px -2px rgb(255 107 0 / 25%);
      }
      .swagger-ui .tab li,
      .swagger-ui .tab li button { color: var(--text-muted); }
      .swagger-ui .tab li.active,
      .swagger-ui .tab li.active button { color: var(--text-primary); }
      .swagger-ui .tab li.active { border-bottom-color: var(--accent); }
      .swagger-ui .responses-table td,
      .swagger-ui .responses-table th,
      .swagger-ui .parameters-col_description,
      .swagger-ui .parameter__extension,
      .swagger-ui .model-box,
      .swagger-ui .model-container { border-color: var(--border-color); }
      .swagger-ui .btn:not(.authorize):not(.execute) {
        border-radius: 6px;
        background: transparent;
        transition: border-color 150ms ease, background 150ms ease, color 150ms ease;
      }
      .swagger-ui .btn:not(.authorize):not(.execute):hover { background: rgb(255 255 255 / 5%); }
      .swagger-ui .btn.authorize { border-radius: 6px; }
      :root[data-theme="light"] .site-header { background: rgb(17 24 39 / 96%); }
      :root[data-theme="light"] .swagger-ui .opblock .opblock-summary-path,
      :root[data-theme="light"] .swagger-ui .opblock .opblock-summary-path__deprecated { color: #1e293b; }
      @media (max-width: 640px) {
        .site-header__content { min-height: 68px; padding: 14px 18px; }
        .site-header__brand { gap: 10px; }
        .theme-toggle { padding: 8px 10px; }
        #swagger-ui { padding: 20px 16px; }
        .swagger-ui .info { margin-top: 0; }
      }
      @media (max-width: 480px) {
        .brand-caption { display: none; }
        .swagger-ui .info .title { font-size: 24px; }
        .swagger-ui .opblock .opblock-summary-method { min-width: 62px; }
      }
    </style>
  </head>
  <body>
    <header class="site-header">
      <div class="site-header__content">
        <div class="site-header__brand">
          <img class="brand-icon" src="/icon_transp.png" alt="CliDoc" />
          <div>
            <p class="brand-name">CliDoc</p>
            <p class="brand-caption">Documentação interativa da API</p>
          </div>
        </div>
        <button class="theme-toggle" id="theme-toggle" type="button" aria-pressed="false">Tema escuro</button>
      </div>
    </header>
    <div id="swagger-ui"></div>
    <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.11.0/swagger-ui-bundle.js"></script>
    <script>
      const themeToggle = document.getElementById('theme-toggle');
      function updateThemeToggle() {
        const isDark = document.documentElement.dataset.theme === 'dark';
        themeToggle.textContent = isDark ? 'Tema claro' : 'Tema escuro';
        themeToggle.setAttribute('aria-pressed', String(isDark));
        themeToggle.setAttribute('aria-label', isDark ? 'Ativar tema claro' : 'Ativar tema escuro');
      }

      updateThemeToggle();
      themeToggle.addEventListener('click', function () {
        const isDark = document.documentElement.dataset.theme !== 'dark';
        document.documentElement.dataset.theme = isDark ? 'dark' : 'light';
        try {
          localStorage.setItem('clidoc-theme', isDark ? 'dark' : 'light');
        } catch (_) {}
        updateThemeToggle();
      });

      window.onload = function () {
        SwaggerUIBundle({
          url: "/openapi.json",
          dom_id: '#swagger-ui',
        });
      };
    </script>
  </body>
</html>`

func normalizePath(p string) string {
	if p == "" {
		return "."
	}
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Clean(filepath.Join(".", p))
}

func ensureDir(path string) {
	if strings.TrimSpace(path) == "" {
		return
	}
	_ = os.MkdirAll(filepath.Dir(normalizePath(path)), 0o755)
}
