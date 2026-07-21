package cluster

import (
	"time"

	cbinsights "github.com/couchbase/gocbinsights"
)

const DefaultHandleTimeout = 10 * time.Second

type Cluster struct {
	*cbinsights.Cluster

	handleTimeout time.Duration
}

func New(sdkCluster *cbinsights.Cluster, handleTimeout time.Duration) *Cluster {
	if handleTimeout == 0 {
		handleTimeout = DefaultHandleTimeout
	}

	return &Cluster{
		Cluster:       sdkCluster,
		handleTimeout: handleTimeout,
	}
}

func (c *Cluster) HandleTimeout() time.Duration {
	if c == nil {
		return 0
	}

	return c.handleTimeout
}
