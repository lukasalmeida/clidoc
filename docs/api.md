# CliDoc API

- Version: 1.0.0
- Base URL: http://localhost:8080

## GET /api/v1/users

**Resumo:** Listar usuários
**Descrição:** Retorna a lista de usuários.
**Tag:** Usuários

### Parâmetros
Nenhum parâmetro informado.

### Respostas
| Status | Tipo | Descrição |
| --- | --- | --- |
| 200 | json | Lista retornada com sucesso |

## POST /api/v1/users

**Resumo:** Criar usuário
**Descrição:** Cria um novo usuário.
**Tag:** Usuários

### Parâmetros
| Nome | Em | Tipo | Obrigatório | Descrição |
| --- | --- | --- | --- | --- |
| id | path | string | sim | ID do usuário |
| role | query | string | não | Papel do usuário |

### Respostas
| Status | Tipo | Descrição |
| --- | --- | --- |
| 201 | json | Usuário criado com sucesso |
| 400 | json | Dados inválidos |

