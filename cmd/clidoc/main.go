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
    <title>CliDoc</title>
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.11.0/swagger-ui.css" />
    <style>
      body { margin: 0; background: #111827; }
      #swagger-ui { max-width: 1200px; margin: 0 auto; padding: 24px; }
    </style>
  </head>
  <body>
    <div id="swagger-ui"></div>
    <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.11.0/swagger-ui-bundle.js"></script>
    <script>
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
