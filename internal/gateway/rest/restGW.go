package rest

import (
	"errors"
	"net/http"

	ctxPkg "github.com/eclipse-xfsc/microservice-core-go/pkg/ctx"

	"github.com/gin-gonic/gin"

	"github.com/eclipse-xfsc/oid4-vci-vp-well-known-service/config"
	"github.com/eclipse-xfsc/oid4-vci-vp-well-known-service/internal/importer"
)

type Gateway struct {
	conf config.GatewayConfig
	imp  importer.Importer
}

func NewGateway(conf config.GatewayConfig, imp importer.Importer) Gateway {
	return Gateway{
		conf: conf,
		imp:  imp,
	}
}

func (gw Gateway) WellKnownCredentialIssuerHandler(c *gin.Context) {
	log := ctxPkg.GetLogger(c)

	tenantId := c.Param("tenantId")
	if tenantId == "" {
		c.JSON(404, "Not found.")
	}

	metadata, err := gw.imp.GetCredentialIssuerMetadata(c, tenantId)

	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, importer.ErrNotFound) {
			status = http.StatusNotFound
		}

		if err := c.AbortWithError(status, err); err != nil {
			log.Error(err, "failed to write status")
		}
	}

	c.JSON(200, metadata)
}
