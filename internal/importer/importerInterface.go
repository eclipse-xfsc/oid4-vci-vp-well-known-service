package importer

import (
	"context"
	"errors"

	"github.com/eclipse-xfsc/oid4-vci-vp-library/model/credential"

	serverPkg "github.com/eclipse-xfsc/microservice-core-go/pkg/server"

	"github.com/eclipse-xfsc/oid4-vci-vp-well-known-service/internal/common"
)

var ErrNotFound = errors.New("not found")

type Importer interface {
	Start(ctx context.Context, server *serverPkg.Server, env *common.Environment) error
	Stop() error
	GotErrors() bool
	GetCredentialIssuerMetadata(ctx context.Context, tenantID string) (*credential.IssuerMetadata, error)
}
