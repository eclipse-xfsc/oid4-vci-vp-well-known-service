package config

import (
	"fmt"
	"log/slog"
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
	CredentialIssuer                  CredentialIssuerConfig        `envconfig:"CREDENTIAL_ISSUER"`
	Gateway                           GatewayConfig                 `envconfig:"GATEWAY"`
	CredentialConfigurationExpiration int                           `envconfig:"CREDENTIAL_CONFIGURATION_EXPIRATION" default:"60"`
}

type GatewayConfig struct {
	LocationHeaderKey string `envconfig:"LOCATION_HEADER_KEY"`
	JwksUrlHeaderKey  string
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

const EnvPrefix = "WELLKNOWN_SERVICE"

func (c *Config) Validate() error {
	var missing []string

	check := func(value, env string) {
		env = EnvPrefix + "_" + env

		if value == "" {
			slog.Warn("environment variable not set", "env", env)
			missing = append(missing, env)
		}
	}

	if c.CredentialIssuer.Importer == ImporterGit {
		check(c.Git.Repo, "GIT_REPO")
		check(c.Git.Token, "GIT_TOKEN")
		check(c.Git.ImagePath, "GIT_IMAGE_PATH")

		if c.Git.Interval == 0 {
			check("", "GIT_INTERVAL")
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %v", missing)
	}

	return nil
}
