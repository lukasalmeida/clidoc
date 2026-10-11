# Contexto do Projeto: CliDoc

## 1. Visão Geral do Projeto

**CliDoc** é uma ferramenta de linha de comando (*CLI*) open source desenvolvida para automatizar a geração de documentação de APIs (especificação OpenAPI 3.0 e Markdown) a partir da análise estática de comentários e código-fonte.

### Objetivo
Eliminar a manutenção manual de documentação de APIs, permitindo que desenvolvedores documentem seus endpoints diretamente no código-fonte de forma declarativa e agnóstica de linguagem.

---

## 2. Arquitetura do Sistema

O CliDoc segue uma arquitetura baseada em pipeline de transformações desacopladas:

```
+------------------+     +------------------------+     +-----------------------------------+
|  Código-Fonte    | --> |  Parsers (AST/Regex)   | --> | Representação Intermediária (IR)  |
| (Go, Python, JS) |     | (Extração de Metadados)|     | (Estrutura Unificada em Memória)  |
+------------------+     +------------------------+     +-----------------------------------+
                                                                          |
                                                                          v
+------------------+     +------------------------+     +-----------------------------------+
| Live Web Server  | <-- | Output File            | <-- | Generators                        |
| (Swagger UI/Redoc|     | (openapi.json / .md)   |     | (OpenAPI 3.0 / Markdown)          |
+------------------+     +------------------------+     +-----------------------------------+
```

### Módulos Principais

1. **Core CLI Engine:** Gerencia os comandos de linha de comando (`init`, `generate`, `serve`), flags e arquivos de configuração (`clidoc.yaml`).
2. **Parser Interface:** Define o contrato padrão que cada analisador de linguagem precisa implementar para extrair comentários e analisar ASTs.
3. **Intermediate Representation (IR):** Estrutura de dados neutra que armazena informações de rotas, parâmetros, respostas e schemas.
4. **Generators:** Converte a IR para formatos específicos (OpenAPI 3.0 JSON/YAML ou Markdown).
5. **Preview Server:** Servidor HTTP embutido para visualização interativa da documentação com suporte a *Hot Reload*.

---

## 3. Especificação do Formato de Anotações

A sintaxe de anotação utilizada nos comentários do código-fonte segue o padrão:

| Anotação | Descrição | Exemplo |
| :--- | :--- | :--- |
| `@Route` | Método HTTP e caminho da rota | `@Route POST /api/v1/users` |
| `@Summary` | Resumo simples da rota | `@Summary Criar Usuário` |
| `@Description` | Descrição detalhada da rota | `@Description Cadastra um novo usuário` |
| `@Tag` | Agrupamento de rotas | `@Tag Usuários` |
| `@Param` | Parâmetros (`<nome> <in> <tipo> <required> "<desc>"`) | `@Param id path string true "ID do usuário"` |
| `@Success` | Resposta de sucesso (`<status_code> <tipo> "<desc>"`) | `@Success 201 json "Criado com sucesso"` |
| `@Failure` | Resposta de erro (`<status_code> <tipo> "<desc>"`) | `@Failure 400 json "Dados inválidos"` |

---

## 4. Representação Intermediária (IR) - Go Data Structure

```go
package ir

type API struct {
	Title       string   `json:"title"`
	Version     string   `json:"version"`
	BaseURL     string   `json:"base_url"`
	Endpoints   []Route  `json:"endpoints"`
}

type Route struct {
	Method      string      `json:"method"`
	Path        string      `json:"path"`
	Summary     string      `json:"summary"`
	Description string      `json:"description"`
	Tag         string      `json:"tag"`
	Params      []Parameter `json:"params"`
	Responses   []Response  `json:"responses"`
}

type Parameter struct {
	Name        string `json:"name"`
	In          string `json:"in"` // path, query, header, body
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Description string `json:"description"`
}

type Response struct {
	StatusCode  int    `json:"status_code"`
	Type        string `json:"type"`
	Description string `json:"description"`
}
```

---

## 5. Roadmap de Desenvolvimento

- [x] **v0.1.0 (MVP):** Parser Go básico via AST e gerador OpenAPI 3.0 JSON.
- [ ] **v0.2.0:** Suporte a exportação em formato Markdown.
- [ ] **v0.3.0:** Adição do comando `clidoc serve` com UI estática do Swagger.
- [ ] **v0.4.0:** Leitura de Schemas/Structs automaticamente para mapeamento de DTOs.
- [ ] **v0.5.0:** Implementação de Parsers para Python (Docstrings) e TypeScript/Node.js (JSDoc).