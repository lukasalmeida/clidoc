package sample

// @Route GET /api/v1/users
// @Summary Listar usuários
// @Description Retorna a lista de usuários.
// @Tag Usuários
// @Success 200 json "Lista retornada com sucesso"
func ListUsers() {}

// @Route POST /api/v1/users
// @Summary Criar usuário
// @Description Cria um novo usuário.
// @Tag Usuários
// @Param id path string true "ID do usuário"
// @Param role query string false "Papel do usuário"
// @Success 201 json "Usuário criado com sucesso"
// @Failure 400 json "Dados inválidos"
func CreateUser(id string) {}
