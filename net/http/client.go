package http

import (
	"net"
	"net/http"
	"time"
)

// DefaultClient is a http.Client with properly configured timeouts.
var DefaultClient = &http.Client{
	Timeout: time.Second * 10,
	Transport: &http.Transport{
		Dial: (&net.Dialer{
			Timeout: time.Second * 5,
		}).Dial,
		TLSHandshakeTimeout: time.Second * 5,
		MaxIdleConns:        100,
	},
}
