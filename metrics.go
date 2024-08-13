package drovedns

import (
	"github.com/coredns/coredns/plugin"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	DroveQueryTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: plugin.Namespace,
		Subsystem: pluginName,
		Name:      "sync_total",
		Help:      "Counter of Drove sync successful",
	})

	DroveQueryFailure = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: plugin.Namespace,
		Subsystem: pluginName,
		Name:      "sync_failure",
		Help:      "Counter of Drove syncs failed",
	})

	DroveApiRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: plugin.Namespace,
		Subsystem: pluginName,
		Name:      "api_total",
		Help:      "Drove api requests total",
	}, []string{"code", "method", "host"})

	DroveControllerHealth = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: plugin.Namespace,
		Subsystem: pluginName,
		Name:      "controller_health",
		Help:      "Drove controller health",
	}, []string{"host"})
)
