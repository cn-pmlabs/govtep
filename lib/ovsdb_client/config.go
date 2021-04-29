package ovsdbclient

import (
	"crypto/tls"
)

// Config db client config
type Config struct {
	Db        string
	Addr      string
	TLSConfig *tls.Config
}
