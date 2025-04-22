package broadcast

import (
	"context"
	"encoding/json"

	"github.com/eclipse-xfsc/oid4-vci-vp-well-known-service/internal/importer"

	messaging "github.com/eclipse-xfsc/nats-message-library"

	"github.com/eclipse-xfsc/oid4-vci-vp-library/model/credential"

	"github.com/eclipse-xfsc/microservice-core-go/pkg/logr"
	"github.com/eclipse-xfsc/microservice-core-go/pkg/server"

	"github.com/cloudevents/sdk-go/v2/event"
	ce "github.com/eclipse-xfsc/cloud-event-provider"
	"golang.org/x/sync/errgroup"

	"github.com/eclipse-xfsc/oid4-vci-vp-well-known-service/internal/common"
	"github.com/eclipse-xfsc/oid4-vci-vp-well-known-service/internal/service"
)

type Importer struct {
	stopChan chan bool

	svc        service.IssuerService
	natsConfig ce.NatsConfig
	log        logr.Logger
}

var _ importer.Importer = &Importer{}

func NewImporter(svc service.IssuerService, natsConfig ce.NatsConfig, logger logr.Logger) *Importer {
	return &Importer{
		stopChan:   make(chan bool),
		svc:        svc,
		natsConfig: natsConfig,
		log:        logger,
	}
}

func (b *Importer) Start(ctx context.Context, _ *server.Server, _ *common.Environment) error {
	errGrp, ctx := errgroup.WithContext(ctx)

	errGrp.Go(func() error {
		return b.listen(ctx)
	})

	return nil
}

func (b *Importer) Stop() error {
	b.stopChan <- true
	return nil
}

func (b *Importer) GotErrors() bool {
	return false
}

func (b *Importer) GetCredentialIssuerMetadata(ctx context.Context, tenantID string) (*credential.IssuerMetadata, error) {
	return b.svc.GetIssuer(ctx, tenantID, false)
}

func (b *Importer) listen(ctx context.Context) error {
	client, err := ce.New(
		ce.Config{Protocol: ce.ProtocolTypeNats, Settings: b.natsConfig},
		ce.ConnectionTypeSub,
		messaging.TopicIssuerRegistration,
	)
	if err != nil {
		return err
	}

	for {
		if err := client.SubCtx(ctx, b.handleEvent); err != nil {
			b.log.Error(err, "cloudEventProvider.Sub failed")
		}
	}
}

// TODO: define events?!
func (b *Importer) handleEvent(e event.Event) {
	switch e.Type() {
	case messaging.EventTypeIssuerRegistration:
		b.handleIssuerEvent(context.TODO(), e.Data())
	default:
		b.log.Info("received unknown event type", "type", e.Type(), "data", e.String())
	}
}

func (b *Importer) handleIssuerEvent(ctx context.Context, data []byte) {
	var msg messaging.IssuerRegistration
	if err := json.Unmarshal(data, &msg); err != nil {
		b.log.Error(err, "failed to unmarshal issuer")
	}

	if msg.TenantId == "" {
		b.log.Info("invalid request.message (empty tenantID)", "msg", msg)
	}

	if err := b.svc.UpsertIssuer(ctx, msg.TenantId, msg.Issuer); err != nil {
		b.log.Error(err, "failed to UpsertIssuer")
	}
}
