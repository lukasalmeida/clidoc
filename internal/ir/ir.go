package ir

// API representa a documentação de uma API.
type API struct {
	Title     string  `json:"title"`
	Version   string  `json:"version"`
	BaseURL   string  `json:"base_url"`
	Endpoints []Route `json:"endpoints"`
}

// Route representa uma rota documentada.
type Route struct {
	Method      string      `json:"method"`
	Path        string      `json:"path"`
	Summary     string      `json:"summary"`
	Description string      `json:"description"`
	Tag         string      `json:"tag"`
	Params      []Parameter `json:"params"`
	Responses   []Response  `json:"responses"`
}

// Parameter representa um parâmetro de rota.
type Parameter struct {
	Name        string `json:"name"`
	In          string `json:"in"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Description string `json:"description"`
}

// Response representa uma resposta da API.
type Response struct {
	StatusCode  int    `json:"status_code"`
	Type        string `json:"type"`
	Description string `json:"description"`
}
