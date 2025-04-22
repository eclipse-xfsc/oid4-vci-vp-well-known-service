package nats

import (
	"context"
	"encoding/json"

	ctxPkg "github.com/eclipse-xfsc/microservice-core-go/pkg/ctx"

	"github.com/cloudevents/sdk-go/v2/event"
	ce "github.com/eclipse-xfsc/cloud-event-provider"
	"github.com/eclipse-xfsc/nats-message-library/common"
	"golang.org/x/sync/errgroup"

	messaging "github.com/eclipse-xfsc/nats-message-library"
	"github.com/eclipse-xfsc/oid4-vci-vp-well-known-service/internal/service"
)

type Gateway struct {
	svc        service.IssuerService
	natsConfig ce.NatsConfig
}

func NewGateway(svc service.IssuerService, config ce.NatsConfig) Gateway {
	return Gateway{
		svc:        svc,
		natsConfig: config,
	}
}

func (gw Gateway) Run(ctx context.Context) error {
	errGrp, ctx := errgroup.WithContext(ctx)

	errGrp.Go(func() error {
		return gw.GetIssuerMetadata(ctx)
	})

	return errGrp.Wait()
}

// GetIssuerMetadata initializes a new cloudeventprovider.CloudEventProviderClient, waits for an
// incoming messaging.TopicGetIssuerMetadata request and replies to it.
// The function is blocking and never returns, once the client was successfully initialized
func (gw Gateway) GetIssuerMetadata(ctx context.Context) error {
	client, err := ce.New(
		ce.Config{
			Protocol: ce.ProtocolTypeNats,
			Settings: gw.natsConfig,
		},
		ce.ConnectionTypeRep,
		messaging.TopicGetIssuerMetadata,
	)
	if err != nil {
		return err
	}

	log := ctxPkg.GetLogger(ctx)

	for {
		if err := client.ReplyCtx(ctx, gw.getIssuerMetadata); err != nil {
			log.Error(err, "error during getIssuerMetadata")
		}
	}
}

func (gw Gateway) getIssuerMetadata(ctx context.Context, event event.Event) (*event.Event, error) {
	var req messaging.GetIssuerMetadataReq
	if err := event.DataAs(&req); err != nil {
		return nil, err
	}

	issuer, err := gw.svc.GetIssuer(ctx, req.TenantId, true)
	if err != nil {
		return nil, err
	}

	output := messaging.GetIssuerMetadataReply{
		Reply: common.Reply{
			TenantId:  req.TenantId,
			RequestId: req.RequestId,
			Error:     nil,
		},
		Issuer: issuer,
	}

	data, err := json.Marshal(output)
	if err != nil {
		return nil, err
	}

	reply, err := ce.NewEvent(messaging.SourceWellKnownService, messaging.TopicGetIssuerMetadata, data)
	if err != nil {
		return nil, err
	}

	return &reply, nil
}
