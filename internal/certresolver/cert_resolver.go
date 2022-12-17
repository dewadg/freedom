package certresolver

import "github.com/dewadg/freedom/internal/config"

type Resolver interface {
	Cert() string
	PrivateKey() string
	Init() error
}

func NewFileResolver(cfg *config.Config) FileResolver {
	return FileResolver{
		cert:       cfg.SSL.Cert,
		privateKey: cfg.SSL.PrivateKey,
	}
}
