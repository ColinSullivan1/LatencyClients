// Copyright 2019 The NATS Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/nats-io/demos/latency-clients/utils"
	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Variables
var (
	debug                 = false
	defaultListenPort     = 8675
	defaultListenAddress  = "0.0.0.0"
	defaultRequestorCount = 50
	defaultSubject        = "demo.requests"
)

// Config holds general configuration for the clients
type config struct {
	Urls    string
	Subject string
	Creds   string
	Tlscert string
	Tlskey  string
	Tlsca   string
}

// ToMillis converts a duration to milliseconds
func ToMillis(d *time.Duration) float64 {
	if d == nil {
		return 0
	}
	// works because time.Nanosecond is 1...
	return float64(d.Nanoseconds() / int64(time.Millisecond))
}

var (
	// Create a summary to track RPC latencies for three
	// distinct services with different latency distributions. These services are
	// differentiated via a "service" label.
	rpcDurations = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "rpc_durations_seconds",
			Help: "RPC latency distributions.",
		},
		[]string{"service"},
	)

	// The same as above, but now as a histogram, and only for the normal
	// distribution. The buckets are targeted to the parameters of the
	// normal distribution, with 20 buckets centered on the mean, each
	// half-sigma wide.
	rpcDurationsHistogram = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "rpc_durations_histogram_seconds",
		Help:    "RPC latency distributions.",
		Buckets: prometheus.ExponentialBuckets(1.0, 2, 16),
	})

	// Requests per second.
	rpcRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "rpc_reqs_count",
		Help: "Requests",
	}, []string{"service"})
)

// Debugf prints to output if we are debugging
func Debugf(format string, v ...interface{}) {
	if debug {
		log.Printf(format, v...)
	}
}

// NOTE: Use tls scheme for TLS, e.g. nats-req -s tls://demo.nats.io:4443 foo hello
func usage() {
	log.Fatalf("Usage: requestor [-s server (%s)] [-p port] [-nr requestor count] <subject> <msg> \n", nats.DefaultURL)
}

// startHTTP configures and starts the HTTP server for applications to poll data from
// this demo application.
func startHTTP(port int) error {
	var hp string
	var err error
	var config *tls.Config

	hp = net.JoinHostPort(defaultListenAddress, strconv.Itoa(port))

	Debugf("listening at http://%s/metrics", hp)
	listener, err := net.Listen("tcp", hp)
	if err != nil {
		log.Fatalf("can't start HTTP listener: %v", err)
		return err
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	srv := &http.Server{
		Addr:           hp,
		Handler:        mux,
		MaxHeaderBytes: 1 << 20,
		TLSConfig:      config,
	}

	sHTTP := listener
	go func() {
		for i := 0; i < 10; i++ {
			var err error
			if err = srv.Serve(sHTTP); err != nil {
				Debugf("Unable to start HTTP server (may already be running): %v", err)
			} else {
				Debugf("Ready to serve Prometheus requests.")
			}
		}
	}()

	return nil
}

func main() {

	var (
		port    int
		subject string
		urls    string
		delay   string
		dly     time.Duration
		err     error
	)

	conf := utils.NewConfig()

	flag.StringVar(&urls, "s", nats.DefaultURL, "The nats server URLs (separated by comma)")
	flag.StringVar(&subject, "subj", defaultSubject, "The subject to make requests on")
	flag.IntVar(&port, "p", defaultListenPort, "The prometheus port to listen on")
	flag.BoolVar(&debug, "debug", false, "Enable debugging")
	flag.StringVar(&conf.Tlscert, "tlscert", "", "Server certificate file (Enables HTTPS)")
	flag.StringVar(&conf.Tlskey, "tlskey", "", "Private key for server certificate (used with HTTPS)")
	flag.StringVar(&conf.Tlsca, "tlscacert", "", "Client certificate CA for verification (used with HTTPS)")
	flag.StringVar(&conf.Creds, "creds", "", "Credentials file")
	flag.StringVar(&delay, "delay", "", "Delay between each request")

	log.SetFlags(0)
	flag.Parse()

	args := flag.Args()
	if len(args) == 1 {
		subject = args[0]
	}

	if delay != "" {
		dly, err = time.ParseDuration(delay)
		if err != nil {
			log.Fatalf("Can't parse delay: %v\n", err)
		}
	}

	log.Printf("Prometheus Port: %d\n", port)
	log.Printf("Server URLs:     %s\n", urls)
	log.Printf("Subject: %s\n", subject)

	if err := startHTTP(port); err != nil {
		log.Fatalf("Couldn't start HTTP server")
	}

	nc, err := nats.Connect(urls, conf.GetClientOptions()...)
	if err != nil {
		log.Fatalf("Can't connect: %v\n", err)
	}

	// Register the summary and the histogram with Prometheus's default registry.
	prometheus.MustRegister(rpcDurations)
	prometheus.MustRegister(rpcDurationsHistogram)
	prometheus.MustRegister(rpcRequests)

	count := int64(0)

	// Start a group of requestors, simulating load.  Each requestor will
	// send a request, then store the duration into Prometheus.
	go func() {
		for true {
			c := atomic.AddInt64(&count, 1)

			// each request has a sequence for tracing
			payload := fmt.Sprintf("request-%d", c)

			// make the request and save the duration
			start := time.Now()
			_, err := nc.Request(subject, []byte(payload), 10*time.Second)

			// if the connection is closed, exit this function.
			if err == nats.ErrConnectionClosed {
				return
			} else if err != nil {
				// trace all other errors
				Debugf("Request error: %v", err)
			}

			d := time.Now().Sub(start)
			millis := ToMillis(&d)

			rpcDurations.WithLabelValues("rpc-demo-req").Observe(millis)
			rpcDurationsHistogram.Observe(millis)
			rpcRequests.WithLabelValues("rpc-demo-req").Inc()

			if dly != 0 {
				time.Sleep(dly)
			}
		}
	}()

	// Exit via the interrupt handler
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	nc.Close()

	fmt.Printf("Exiting...\n")
}
