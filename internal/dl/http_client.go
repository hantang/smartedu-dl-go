package dl

import (
	"net/http"
	"time"
)

var defaultHTTPClient = &http.Client{
	Timeout: 60 * time.Second,
}
