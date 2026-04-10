package middleware

import (
	"k8s.io/utils/lru"
)

var lruCache = lru.New(1)
