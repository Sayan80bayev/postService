package metrics

import "github.com/prometheus/client_golang/prometheus"

var HttpRequests = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests per route",
	},
	[]string{"route"},
)

func Init() {
	prometheus.MustRegister(HttpRequests)
}
