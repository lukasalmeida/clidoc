# CliDoc

CliDoc é uma ferramenta de linha de comando open source para automatizar a geração de documentação de APIs a partir da análise estática de comentários e código-fonte.

Status: em desenvolvimento — a primeira release está sendo preparada.

## Visão geral

O objetivo do CliDoc é eliminar a manutenção manual de documentação de endpoints, permitindo que desenvolvedores descrevam rotas diretamente no código, de forma declarativa e agnóstica a linguagem.

A ferramenta extrai metadados de comentários e estruturas do código, converte esses dados em uma representação intermediária (IR) e gera documentação em formatos amplamente utilizados, como OpenAPI 3.0 e Markdown.

## O que o CliDoc oferece

- Extração automática de informações de rotas, parâmetros e respostas
- Suporte a geração de documentação OpenAPI 3.0
- Exportação para Markdown
- Estrutura modular para suporte a múltiplas linguagens
- Preview local da documentação com visualização interativa
- Fluxo baseado em pipeline de transformação, facilitando extensão futura

## Arquitetura

O projeto foi pensado em torno de um pipeline de processamento desacoplado:

```text
Código-fonte
   ↓
Parsers / extração de metadados
   ↓
Representação intermediária (IR)
   ↓
Generators
   ↓
OpenAPI / Markdown / Preview
```

### Módulos previstos

- Core CLI Engine: gerenciamento dos comandos e configuração
- Parser Interface: contrato para extração em diferentes linguagens
- Intermediate Representation (IR): estrutura comum para rotas e respostas
- Generators: conversão para OpenAPI e Markdown
- Preview Server: visualização local da documentação

## Formato de anotações

Os comentários do código podem seguir uma convenção declarativa com anotações como as abaixo:

```go
// @Route POST /api/v1/users
// @Summary Criar Usuário
// @Description Cadastra um novo usuário
// @Tag Usuários
// @Param id path string true "ID do usuário"
// @Success 201 json "Criado com sucesso"
// @Failure 400 json "Dados inválidos"
```

### Anotações suportadas

| Anotação | Descrição |
| :--- | :--- |
| `@Route` | Define o método HTTP e o caminho da rota |
| `@Summary` | Resumo da rota |
| `@Description` | Descrição detalhada |
| `@Tag` | Agrupamento da rota |
| `@Param` | Parâmetro da rota, query, header ou body |
| `@Success` | Resposta de sucesso |
| `@Failure` | Resposta de erro |

## Exemplo de estrutura de representação intermediária

```go
package ir

type API struct {
    Title     string  `json:"title"`
    Version   string  `json:"version"`
    BaseURL   string  `json:"base_url"`
    Endpoints []Route `json:"endpoints"`
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
```

## Roadmap

- [x] MVP inicial com parser Go básico e geração de OpenAPI 3.0 JSON
- [ ] Suporte a exportação em Markdown
- [ ] Comando de preview local da documentação
- [ ] Leitura automática de schemas e DTOs
- [ ] Suporte a Python e TypeScript/Node.js

## Status da implementação

O projeto está em fase inicial de desenvolvimento. A base da arquitetura e o modelo de anotações já foram definidos, e a primeira release será focada em entregar a geração de documentação de forma simples, automatizada e extensível.

## Como usar (em breve)

A CLI ainda está sendo implementada. O fluxo planejado será:

```bash
clidoc init
clidoc generate
clidoc serve
```

Esses comandos terão como objetivo inicializar a configuração, gerar a documentação a partir do código e abrir uma visualização local da API documentada.

## Ambiente de desenvolvimento

O ambiente de desenvolvimento usa Docker Compose e Go. É necessário ter Docker com o plugin Docker Compose instalado.

Inicie o contêiner de desenvolvimento na raiz do repositório:

```bash
docker compose up -d --build
docker compose exec dev sh
```

O código do repositório fica disponível em `/workspace`. As dependências e o cache de compilação do Go são mantidos em volumes Docker entre reinicializações. A porta `8080` do host é encaminhada para o contêiner para uso futuro pelo servidor de preview.

Para parar o ambiente:

```bash
docker compose down
```

## Contribuição

Contribuições são bem-vindas. O projeto ainda está em desenvolvimento e a estrutura da base pode evoluir conforme a primeira release for amadurecida.

## Licença

O projeto é licenciado sob a Apache License 2.0 — veja [LICENSE.md](LICENSE.md).
