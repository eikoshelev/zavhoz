package main

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	totalRequestDns = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "DNS",
		Help: "Total DNS requests",
	}, []string{"rcode"})

	totalRequestHttp = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "HTTP",
		Help: "Total HTTP requests",
	}, []string{"status"})
)

func init() {

	prometheus.MustRegister(totalRequestDns)
	prometheus.MustRegister(totalRequestHttp)
}
