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

package utils

import (
	"fmt"
	"log"
	"os"

	"github.com/nats-io/nats.go"
)

// Config holds shared configuration for the clients
type Config struct {
	Creds   string
	Tlscert string
	Tlskey  string
	Tlsca   string
}

// NewConfig returns a new configuration struct
func NewConfig() *Config {
	return &Config{}
}

// GetClientOptions cenerates NATS client options.
func (conf *Config) GetClientOptions() []nats.Option {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "127.0.0.1"
	}
	opts := []nats.Option{nats.Name(fmt.Sprintf("NATS_Requestor - %s", hostname))}

	if conf.Creds != "" {
		opts = append(opts, nats.UserCredentials(conf.Creds))
	}
	opts = append(opts, nats.DisconnectHandler(func(_ *nats.Conn) {
		log.Println("Disconnected")
	}))
	opts = append(opts, nats.ReconnectHandler(func(c *nats.Conn) {
		log.Printf("Reconnected to %v", c.ConnectedAddr())
	}))
	opts = append(opts, nats.ClosedHandler(func(_ *nats.Conn) {
		log.Println("Connection closed")
	}))
	opts = append(opts, nats.ErrorHandler(func(c *nats.Conn, s *nats.Subscription, err error) {
		if s != nil {
			log.Printf("Error: err=%v", err)
		} else {
			log.Printf("Error: subject=%s, err=%v", s.Subject, err)
		}
	}))
	opts = append(opts, nats.MaxReconnects(10240))
	if conf.Tlsca != "" {
		opts = append(opts, nats.RootCAs(conf.Tlsca))
	}
	if conf.Tlscert != "" {
		opts = append(opts, nats.ClientCert(conf.Tlscert, conf.Tlskey))
	}
	return opts
}
