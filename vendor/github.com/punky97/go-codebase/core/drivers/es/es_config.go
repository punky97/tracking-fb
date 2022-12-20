package es

import (
	"time"

	"github.com/spf13/viper"
)

// Connection -- ES connection
type Connection struct {
	host       string
	port       int
	sniff      bool
	timeout    time.Duration // in seconds
	apmTracing bool
}

// DefaultEsConnectionFromConfig -- Load connection settings
func DefaultEsConnectionFromConfig() *Connection {
	return &Connection{
		host:       viper.GetString("elasticsearch.host"),
		port:       viper.GetInt("elasticsearch.port"),
		timeout:    viper.GetDuration("elasticsearch.timeout"),
		sniff:      viper.GetBool("elasticsearch.sniff"),
		apmTracing: viper.GetBool("elasticsearch_apm.enable"),
	}
}
