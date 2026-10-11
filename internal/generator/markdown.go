package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lukasalmeida/clidoc/internal/ir"
)

func GenerateMarkdown(api ir.API) string {
	if api.Title == "" {
		api.Title = "CliDoc API"
	}
	if api.Version == "" {
		api.Version = "1.0.0"
	}

	var sb strings.Builder
	sb.WriteString("# ")
	sb.WriteString(api.Title)
	sb.WriteString("\n\n")
	sb.WriteString("- Version: ")
	sb.WriteString(api.Version)
	sb.WriteString("\n")
	if api.BaseURL != "" {
		sb.WriteString("- Base URL: ")
		sb.WriteString(api.BaseURL)
		sb.WriteString("\n")
	}
	sb.WriteString("\n")

	if len(api.Endpoints) == 0 {
		sb.WriteString("Nenhuma rota documentada.")
		return sb.String()
	}

	for _, route := range api.Endpoints {
		sb.WriteString("## ")
		sb.WriteString(strings.ToUpper(route.Method))
		sb.WriteString(" ")
		sb.WriteString(route.Path)
		sb.WriteString("\n\n")
		if route.Summary != "" {
			sb.WriteString("**Resumo:** ")
			sb.WriteString(route.Summary)
			sb.WriteString("\n")
		}
		if route.Description != "" {
			sb.WriteString("**Descrição:** ")
			sb.WriteString(route.Description)
			sb.WriteString("\n")
		}
		if route.Tag != "" {
			sb.WriteString("**Tag:** ")
			sb.WriteString(route.Tag)
			sb.WriteString("\n")
		}
		sb.WriteString("\n")

		sb.WriteString("### Parâmetros\n")
		if len(route.Params) == 0 {
			sb.WriteString("Nenhum parâmetro informado.\n\n")
		} else {
			sb.WriteString("| Nome | Em | Tipo | Obrigatório | Descrição |\n")
			sb.WriteString("| --- | --- | --- | --- | --- |\n")
			for _, param := range route.Params {
				sb.WriteString("| ")
				sb.WriteString(param.Name)
				sb.WriteString(" | ")
				sb.WriteString(param.In)
				sb.WriteString(" | ")
				sb.WriteString(param.Type)
				sb.WriteString(" | ")
				sb.WriteString(boolLabel(param.Required))
				sb.WriteString(" | ")
				sb.WriteString(param.Description)
				sb.WriteString(" |\n")
			}
			sb.WriteString("\n")
		}

		sb.WriteString("### Respostas\n")
		if len(route.Responses) == 0 {
			sb.WriteString("Nenhuma resposta documentada.\n\n")
		} else {
			sb.WriteString("| Status | Tipo | Descrição |\n")
			sb.WriteString("| --- | --- | --- |\n")
			for _, resp := range route.Responses {
				sb.WriteString("| ")
				sb.WriteString(fmt.Sprintf("%d", resp.StatusCode))
				sb.WriteString(" | ")
				sb.WriteString(resp.Type)
				sb.WriteString(" | ")
				sb.WriteString(resp.Description)
				sb.WriteString(" |\n")
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

func WriteMarkdown(path string, api ir.API) error {
	content := GenerateMarkdown(api)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write markdown: %w", err)
	}
	return nil
}

func boolLabel(v bool) string {
	if v {
		return "sim"
	}
	return "não"
}
