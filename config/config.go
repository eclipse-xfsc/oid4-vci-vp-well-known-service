package config

import (
	"time"

	cfgPkg "github.com/eclipse-xfsc/microservice-core-go/pkg/config"
	postgresPkg "github.com/eclipse-xfsc/microservice-core-go/pkg/db/postgres"

	cloudeventprovider "github.com/eclipse-xfsc/cloud-event-provider"
)

const (
	ImporterGit       = "GIT"
	ImporterBroadcast = "BROADCAST"
)

type Config struct {
	cfgPkg.BaseConfig `envconfig:"CORE"`

	Postgres                          postgresPkg.Config            `envconfig:"POSTGRES"`
	Nats                              cloudeventprovider.NatsConfig `envconfig:"NATS"`
	Git                               GitConfig                     `envconfig:"GIT"`
	JwtIssuer                         JwtIssuerConfig               `envconfig:"OPEN_ID"`
	CredentialIssuer                  CredentialIssuerConfig        `envconfig:"CREDENTIAL_ISSUER"`
	Gateway                           GatewayConfig                 `envconfig:"GATEWAY"`
	CredentialConfigurationExpiration int                           `envconfig:"CREDENTIAL_CONFIGURATION_EXPIRATION" default:"60"`
}

type GatewayConfig struct {
	LocationHeaderKey string `envconfig:"LOCATION_HEADER_KEY"`
	JwksUrlHeaderKey  string
}

type JwtIssuerConfig struct {
	Issuer string `envconfig:"ISSUER"`
}

type CredentialIssuerConfig struct {
	Importer string `envconfig:"IMPORTER" required:"true" default:"BROADCAST"`
}

type GitConfig struct {
	ImagePath string        `envconfig:"IMAGE_PATH"`
	Repo      string        `envconfig:"REPO"`
	Token     string        `envconfig:"TOKEN"`
	Interval  time.Duration `envconfig:"INTERVAL"`
}
