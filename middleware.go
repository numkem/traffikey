package traffikey

type Middleware struct {
	Name   string            `json:"name"`
	Kind   string            `json:"kind"`
	Values map[string]string `json:"values"`
}
