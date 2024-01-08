package traffikey

type Target struct {
	Name         string            `json:"name"`
	Type         string            `json:"type"`
	ServerURLs   []string          `json:"urls"`
	Entrypoint   string            `json:"entrypoint"`
	Middlewares  []*Middleware     `json:"middlewares"`
	Prefix       string            `json:"prefix"`
	Rule         string            `json:"rule"`
	TLS          bool              `json:"tls"`
	TLSExtraKeys map[string]string `json:"tls_extra_keys"`
}
