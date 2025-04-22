package config

import cloudeventprovider "github.com/eclipse-xfsc/cloud-event-provider"

type Config struct {
	Nats cloudeventprovider.NatsConfig `envconfig:"NATS"`
}
