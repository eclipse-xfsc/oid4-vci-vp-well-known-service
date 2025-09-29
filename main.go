package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	postgresPkg "github.com/eclipse-xfsc/microservice-core-go/pkg/db/postgres"
	errPkg "github.com/eclipse-xfsc/microservice-core-go/pkg/err"
	"github.com/eclipse-xfsc/microservice-core-go/pkg/logr"
	serverPkg "github.com/eclipse-xfsc/microservice-core-go/pkg/server"

	"github.com/gin-gonic/gin"
	"github.com/kelseyhightower/envconfig"
	"golang.org/x/sync/errgroup"

	"github.com/eclipse-xfsc/oid4-vci-vp-well-known-service/config"
	"github.com/eclipse-xfsc/oid4-vci-vp-well-known-service/internal/common"
	pgIssuers "github.com/eclipse-xfsc/oid4-vci-vp-well-known-service/internal/database/issuers/postgres"
	"github.com/eclipse-xfsc/oid4-vci-vp-well-known-service/internal/database/postgres"
	"github.com/eclipse-xfsc/oid4-vci-vp-well-known-service/internal/gateway/nats"
	"github.com/eclipse-xfsc/oid4-vci-vp-well-known-service/internal/gateway/rest"
	"github.com/eclipse-xfsc/oid4-vci-vp-well-known-service/internal/importer"
	"github.com/eclipse-xfsc/oid4-vci-vp-well-known-service/internal/importer/broadcast"
	"github.com/eclipse-xfsc/oid4-vci-vp-well-known-service/internal/importer/git"
	"github.com/eclipse-xfsc/oid4-vci-vp-well-known-service/internal/service"
)

var env *common.Environment
var conf config.Config

func main() {
	errGrp, ctx := errgroup.WithContext(context.Background())

	if err := envconfig.Process("WELLKNOWN_SERVICE", &conf); err != nil {
		panic(fmt.Sprintf("failed to load config from env: %+v", err))
	}

	logger, err := logr.New(conf.LogLevel, conf.BaseConfig.IsDev, nil)
	if err != nil {
		log.Fatalf("failed to init logger: %t", err)
	}

	errChan := make(chan error)
	go errPkg.LogChan(*logger, errChan)
	pgDb, err := postgresPkg.ConnectRetry(ctx, conf.Postgres, time.Minute, errChan)
	if err != nil {
		logger.Error(err, "failed to connect to postgres")
		os.Exit(1)
	}

	if err := postgresPkg.MigrateUP(pgDb, postgres.Migrations, "migrations"); err != nil {
		logger.Error(err, fmt.Sprintf("migration failed: %s", err))
		os.Exit(1)
	}

	issuerSvc := service.NewIssuerService(pgIssuers.NewStore(pgDb, *logger, conf))

	var imp importer.Importer
	switch conf.CredentialIssuer.Importer {
	case config.ImporterGit:
		imp = git.NewImporter(conf.Git, *logger)
	case config.ImporterBroadcast:
		imp = broadcast.NewImporter(issuerSvc, conf.Nats, *logger)
	default:
		panic("no importer defined")
	}

	env = common.GetEnvironment()
	env.SetLogger(logger)
	env.SetConfig(&conf)
	env.SetHealthFunc(imp.GotErrors)

	logger.Debug("starting rest server")

	server := serverPkg.New(env)
	if err := imp.Start(ctx, server, env); err != nil {
		logger.Error(err, "Importer cant be started")
	}
	defer imp.Stop()

	restGW := rest.NewGateway(conf.Gateway, imp)

	server.Add(func(rg *gin.RouterGroup) {
		wk := rg.Group("/.well-known")
		wk.GET("/openid-credential-issuer", restGW.WellKnownCredentialIssuerHandler)
	})

	errGrp.Go(func() error {
		return server.Run(conf.ListenPort, conf.ListenAddr)
	})

	logger.Debug("starting nats listener")

	natsGW := nats.NewGateway(issuerSvc, conf.Nats)
	errGrp.Go(func() error {
		return natsGW.Run(ctx)
	})

	if err := errGrp.Wait(); err != nil {
		logger.Error(err, "error during execution")
	}
}
