package certresolver

import "errors"

type FileResolver struct {
	cert       string
	privateKey string
}

func (f FileResolver) Cert() string {
	return f.cert
}

func (f FileResolver) PrivateKey() string {
	return f.privateKey
}

func (f FileResolver) Init() error {
	if f.cert == "" {
		return errors.New("missing certificate")
	}
	if f.privateKey == "" {
		return errors.New("missing private key")
	}
	return nil
}
