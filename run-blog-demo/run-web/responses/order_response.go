package responses

type OrderResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Data    string `json:"data"`
}

type ReadOrdersResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Data    []byte `json:"data"`
}
