package rest

import (
	"errors"
	"net/http"

	ctxPkg "github.com/eclipse-xfsc/microservice-core-go/pkg/ctx"
	"github.com/eclipse-xfsc/oid4-vci-vp-library/model/credential"

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

func (gw Gateway) enrichCredentialIssuerMetadataFromHeaders(
	c *gin.Context,
	metadata *credential.IssuerMetadata,
) {
	if key := gw.conf.CredentialIssuerHeaderKey; key != "" {
		if value := c.GetHeader(key); value != "" {
			metadata.CredentialIssuer = value
		}
	}

	if key := gw.conf.AuthorizationServerHeaderKey; key != "" {
		if value := c.GetHeader(key); value != "" {
			metadata.AuthorizationServers = appendUnique(
				metadata.AuthorizationServers,
				value,
			)
		}
	}

	if key := gw.conf.CredentialEndpointHeaderKey; key != "" {
		if value := c.GetHeader(key); value != "" {
			metadata.CredentialEndpoint = value
		}
	}

	if key := gw.conf.BatchCredentialEndpointHeaderKey; key != "" {
		if value := c.GetHeader(key); value != "" {
			metadata.BatchCredentialEndpoint = stringPtr(value)
		}
	}

	if key := gw.conf.DeferredCredentialEndpointHeaderKey; key != "" {
		if value := c.GetHeader(key); value != "" {
			metadata.DeferredCredentialEndpoint = stringPtr(value)
		}
	}

	if key := gw.conf.NotificationEndpointHeaderKey; key != "" {
		if value := c.GetHeader(key); value != "" {
			metadata.NotificationEndpoint = stringPtr(value)
		}
	}
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}

	return append(values, value)
}

func stringPtr(value string) *string {
	return &value
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

	gw.enrichCredentialIssuerMetadataFromHeaders(c, metadata)

	c.JSON(200, metadata)
}
