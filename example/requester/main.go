package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	cloudeventprovider "github.com/eclipse-xfsc/cloud-event-provider"
	messaging "github.com/eclipse-xfsc/nats-message-library"
	"github.com/eclipse-xfsc/nats-message-library/common"
	"github.com/eclipse-xfsc/oid4-vci-vp-well-known-service/example/issuer/config"
	"github.com/google/uuid"
	"github.com/kelseyhightower/envconfig"
)

var conf config.Config

func main() {
	if err := envconfig.Process("", &conf); err != nil {
		panic(fmt.Sprintf("failed to load config from env: %+v", err))
	}
	conf.Nats.TimeoutInSec = time.Hour
	client, err := cloudeventprovider.New(
		cloudeventprovider.Config{Protocol: cloudeventprovider.ProtocolTypeNats, Settings: conf.Nats},
		cloudeventprovider.ConnectionTypeReq,
		messaging.TopicGetIssuerMetadata,
	)
	if err != nil {
		panic(err)
	}

	interval := time.NewTicker(time.Second * 5)

	data, err := json.Marshal(req)
	if err != nil {
		panic(err)
	}

	event, err := cloudeventprovider.NewEvent("test-client", messaging.EventTypeGetIssuerMetadata, data)
	if err != nil {
		panic(err)
	}

	ctx, _ := context.WithDeadline(context.Background(), time.Now().Add(time.Hour))
	for {
		<-interval.C
		repl, err := client.RequestCtx(ctx, event)
		if err != nil || repl == nil {
			log.Printf("%+v", err)
			continue
		}
		var metadata messaging.GetIssuerMetadataReply
		err = json.Unmarshal(repl.DataEncoded, &metadata)

		log.Printf("send event '%v', got reply: %+v", req, repl.String())
	}
}

func strPtr(s string) *string {
	return &s
}

var req = messaging.GetIssuerMetadataReq{
	Request: common.Request{
		TenantId:  "tenant_space",
		RequestId: uuid.NewString(),
	},
	Format: strPtr("jwt_vc_json"),
}
