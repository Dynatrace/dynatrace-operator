package middleware

import (
	"net/http"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
)

var hashicorpCache = expirable.NewLRU[string, *http.Response](0, nil, time.Hour)
