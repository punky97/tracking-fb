package es

import (
	"github.com/punky97/go-codebase/core/logger"
	"fmt"
	"time"

	"github.com/olivere/elastic/v7"
)

// BkEs -- es
type BkEs struct {
	*elastic.Client
}

// NewEsConnection -- Open connection
func NewConnection(conn *Connection) (*BkEs, error) {
	url := fmt.Sprintf("http://%v:%v", conn.host, conn.port)
	client := &elastic.Client{}
	var err error

	if conn.apmTracing {
		client, err = elastic.NewClient(
			elastic.SetURL(url),
			elastic.SetSnifferTimeoutStartup(time.Second*conn.timeout),
			elastic.SetSniff(conn.sniff),
		)
	} else {
		client, err = elastic.NewClient(
			elastic.SetURL(url),
			elastic.SetSnifferTimeoutStartup(time.Second*conn.timeout),
			elastic.SetSniff(conn.sniff),
		)
	}

	if err != nil {
		logger.BkLog.Fatalf(
			"Cannot create client connection to elasticsearch `%v:%d` with timeout %v with error %v",
			conn.host, conn.port, conn.timeout, err,
		)
	}

	return &BkEs{client}, nil
}
