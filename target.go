package traefikkeymate

type Target struct {
	Name        string        `json:"name"`
	ServerURLs  []string      `json:"urls"`
	Entrypoint  string        `json:"entrypoint"`
	Middlewares []*Middleware `json:"middlewares"`
	Prefix      string        `json:"prefix"`
	Rule        string        `json:"rule"`
	TLS         bool          `json:"tls"`
}
