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
    <meta name="theme-color" content="#10243a" />
    <title>CliDoc | Documentação da API</title>
    <link rel="icon" type="image/png" href="/icon.png" />
    <script>
      try {
        if (localStorage.getItem('clidoc-theme') === 'dark') {
          document.documentElement.dataset.theme = 'dark';
        }
      } catch (_) {}
    </script>
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.11.0/swagger-ui.css" />
    <style>
      :root {
        color-scheme: light;
        font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
        background: #f3f7fb;
        color: #193047;
        --page-bg: #f3f7fb;
        --surface-bg: #fff;
        --surface-muted: #f8fafc;
        --text-primary: #193047;
        --text-muted: #52677b;
        --border-color: #dce6ee;
        --accent: #f97316;
        --accent-hover: #ea580c;
        --shadow: 0 5px 18px rgb(16 36 58 / 5%);
      }
      :root[data-theme="dark"] {
        color-scheme: dark;
        background: #111827;
        color: #e5edf6;
        --page-bg: #111827;
        --surface-bg: #1f2937;
        --surface-muted: #273548;
        --text-primary: #e5edf6;
        --text-muted: #b3c1d1;
        --border-color: #3a4a5e;
        --accent: #fb923c;
        --accent-hover: #fdba74;
        --shadow: 0 5px 18px rgb(0 0 0 / 18%);
      }
      * { box-sizing: border-box; }
      body { margin: 0; background: var(--page-bg); color: var(--text-primary); }
      .site-header {
        background: #10243a;
        border-bottom: 3px solid #f97316;
        color: #fff;
      }
      .site-header__content {
        max-width: 1200px;
        min-height: 76px;
        margin: 0 auto;
        padding: 16px 28px;
        display: flex;
        align-items: center;
        gap: 14px;
      }
      .site-header__brand { display: flex; align-items: center; gap: 14px; }
      .theme-toggle {
        margin-left: auto;
        padding: 9px 13px;
        border: 1px solid #74879b;
        border-radius: 8px;
        background: transparent;
        color: #fff;
        font: inherit;
        font-size: 13px;
        cursor: pointer;
      }
      .theme-toggle:hover { border-color: #fb923c; color: #fdba74; }
      .theme-toggle:focus-visible { outline: 2px solid #fb923c; outline-offset: 3px; }
      .brand-icon { width: 48px; height: 48px; border-radius: 12px; object-fit: cover; }
      .brand-name { margin: 0; font-size: 19px; font-weight: 700; letter-spacing: -.03em; }
      .brand-caption { margin: 3px 0 0; color: #b7c8d8; font-size: 13px; }
      #swagger-ui { max-width: 1200px; margin: 0 auto; padding: 28px; }
      .swagger-ui { color: var(--text-primary); }
      .swagger-ui .topbar { display: none; }
      .swagger-ui .info { margin: 8px 0 28px; }
      .swagger-ui .info .title { color: var(--text-primary); font-size: 30px; }
      .swagger-ui .info p,
      .swagger-ui .info li,
      .swagger-ui .markdown p,
      .swagger-ui .markdown li { color: var(--text-muted); }
      .swagger-ui .info a,
      .swagger-ui .markdown a { color: var(--accent-hover); }
      .swagger-ui .scheme-container {
        padding: 18px 20px;
        border: 1px solid var(--border-color);
        border-radius: 12px;
        background: var(--surface-bg);
        box-shadow: var(--shadow);
      }
      .swagger-ui .opblock-tag {
        color: var(--text-primary);
        border-bottom-color: var(--border-color);
        font-size: 18px;
      }
      .swagger-ui .opblock {
        border-radius: 9px;
        box-shadow: var(--shadow);
      }
      .swagger-ui .opblock .opblock-summary { border-radius: 9px; }
      .swagger-ui .btn.authorize {
        border-color: var(--accent);
        color: var(--accent-hover);
      }
      .swagger-ui .btn.authorize svg { fill: var(--accent-hover); }
      .swagger-ui .btn.execute {
        border-color: var(--accent);
        background: var(--accent);
        color: #fff;
      }
      .swagger-ui .btn.execute:hover { border-color: var(--accent-hover); background: var(--accent-hover); }
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
      @media (max-width: 640px) {
        .site-header__content { min-height: 68px; padding: 14px 18px; }
        .site-header__brand { gap: 10px; }
        .theme-toggle { padding: 8px 10px; }
        #swagger-ui { padding: 20px 16px; }
        .swagger-ui .info { margin-top: 0; }
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
