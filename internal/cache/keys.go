package cache

import (
	"time"
)

const (
	CacheKeyPostsList = "posts:list"
	CacheKeyPostFmt   = "post:%s"
	CacheTTL          = 10 * time.Minute
)
