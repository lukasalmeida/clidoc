package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lukasalmeida/clidoc/internal/ir"
)

func TestGenerateMarkdown(t *testing.T) {
	api := ir.API{
		Title:   "Sample API",
		Version: "1.0.0",
		BaseURL: "http://localhost:8080",
		Endpoints: []ir.Route{{
			Method:      "GET",
			Path:        "/api/v1/users",
			Summary:     "Listar usuários",
			Description: "Retorna a lista de usuários.",
			Tag:         "Usuários",
			Responses: []ir.Response{{
				StatusCode:  200,
				Type:        "json",
				Description: "Lista retornada com sucesso",
			}},
		}},
	}

	content := GenerateMarkdown(api)
	if !strings.Contains(content, "# Sample API") {
		t.Fatalf("expected markdown header")
	}
	if !strings.Contains(content, "## GET /api/v1/users") {
		t.Fatalf("expected route section")
	}
	if !strings.Contains(content, "| 200 | json | Lista retornada com sucesso |") {
		t.Fatalf("expected markdown response table")
	}
}

func TestWriteMarkdown(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "docs", "api.md")
	api := ir.API{Title: "Test API", Version: "1.0.0", Endpoints: []ir.Route{{Method: "POST", Path: "/users"}}}
	if err := WriteMarkdown(out, api); err != nil {
		t.Fatalf("WriteMarkdown returned error: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read markdown output: %v", err)
	}
	if !strings.Contains(string(data), "## POST /users") {
		t.Fatalf("expected generated markdown content")
	}
}
