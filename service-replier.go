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
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/nats-io/demos/latency-clients/utils"
	"github.com/nats-io/nats.go"
)

// Variables
var (
	debug             = false
	defaultSubject    = "demo.requests"
	defaultQueueGroup = "demo"
)

// Debugf prints to output if we are debugging
func Debugf(format string, v ...interface{}) {
	if debug {
		log.Printf(format, v...)
	}
}

func main() {
	var (
		urls    string
		subject string
		qgroup  string
		delay   string
		dly     time.Duration
		err     error
	)

	conf := utils.NewConfig()

	flag.StringVar(&urls, "s", nats.DefaultURL, "The nats server URLs (separated by comma)")
	flag.StringVar(&subject, "subj", defaultSubject, "The subject to listen to")
	flag.StringVar(&qgroup, "qg", defaultQueueGroup, "The name of the queue group")
	flag.StringVar(&delay, "delay", "", "Artificial workload delay")
	flag.BoolVar(&debug, "debug", false, "Enable debugging")
	flag.StringVar(&conf.Tlscert, "tlscert", "", "Server certificate file (Enables HTTPS)")
	flag.StringVar(&conf.Tlskey, "tlskey", "", "Private key for server certificate (used with HTTPS)")
	flag.StringVar(&conf.Tlsca, "tlscacert", "", "Client certificate CA for verification (used with HTTPS)")
	flag.StringVar(&conf.Creds, "creds", "", "Credentials file")

	log.SetFlags(0)
	flag.Parse()

	if delay != "" {
		dly, err = time.ParseDuration(delay)
		if err != nil {
			log.Fatalf("Can't parse delay: %v\n", err)
		}
	}

	nc, err := nats.Connect(urls, conf.GetClientOptions()...)
	if err != nil {
		log.Fatalf("Can't connect: %v\n", err)
	}

	_, err = nc.QueueSubscribe(subject, qgroup, func(msg *nats.Msg) {
		if dly != 0 {
			time.Sleep(dly)
		}
		nc.Publish(msg.Reply, nil)
		Debugf("received: %s\n", string(msg.Data))
	})
	if err != nil {
		log.Fatalf("couldn't subscribe: %v", err)
	}

	// Setup the interrupt handler to drain so we don't miss
	// requests when scaling down.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	Debugf("Draining connection.")
	nc.Drain()
	fmt.Printf("Exiting.\n")
}
