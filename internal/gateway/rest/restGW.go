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
	if metadata == nil {
		return
	}

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

	if key := gw.conf.NonceEndpointHeaderKey; key != "" {
		if value := c.GetHeader(key); value != "" {
			metadata.NonceEndpoint = stringPtr(value)
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

	tenantID := c.Param("tenantId")
	if tenantID == "" {
		c.JSON(
			http.StatusNotFound,
			gin.H{
				"error": "not_found",
			},
		)
		return
	}

	metadata, err := gw.imp.GetCredentialIssuerMetadata(
		c,
		tenantID,
	)
	if err != nil {
		status := http.StatusInternalServerError

		if errors.Is(err, importer.ErrNotFound) {
			status = http.StatusNotFound
		}

		if err := c.AbortWithError(status, err); err != nil {
			log.Error(
				err,
				"failed to write status",
			)
		}

		return
	}

	if metadata == nil {
		c.JSON(
			http.StatusNotFound,
			gin.H{
				"error": "not_found",
			},
		)
		return
	}

	gw.enrichCredentialIssuerMetadataFromHeaders(
		c,
		metadata,
	)

	c.JSON(
		http.StatusOK,
		metadata,
	)
}
