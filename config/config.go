package config

type Config struct {
	Endpoints []Endpoint `json:"endpoints"`
}

type Endpoint struct {
	Method   string      `json:"method"`
	Path     string      `json:"path"`
	Response interface{} `json:"response"`
}
