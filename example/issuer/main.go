package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/eclipse-xfsc/oid4-vci-vp-library/model/credential"

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

	client, err := cloudeventprovider.New(
		cloudeventprovider.Config{Protocol: cloudeventprovider.ProtocolTypeNats, Settings: conf.Nats},
		cloudeventprovider.ConnectionTypePub,
		messaging.TopicIssuerRegistration,
	)
	if err != nil {
		panic(err)
	}

	interval := time.NewTicker(time.Second * 5)

	data, err := json.Marshal(registration)
	if err != nil {
		panic(err)
	}

	event, err := cloudeventprovider.NewEvent("test-client", messaging.EventTypeIssuerRegistration, data)
	if err != nil {
		panic(err)
	}

	for {
		<-interval.C

		if err := client.Pub(event); err != nil {
			log.Printf("%+v", err)
			continue
		}

		log.Printf("send event: %s", registration.Issuer.CredentialIssuer)
	}
}

func strPtr(s string) *string {
	return &s
}

var registration = messaging.IssuerRegistration{
	Request: common.Request{
		TenantId:  "tenant_space",
		RequestId: uuid.NewString(),
	},
	Issuer: credential.IssuerMetadata{
		CredentialIssuer:           "https://cloud-wallet.xfsc.dev",
		AuthorizationServers:       []string{"https://auth-cloud-wallet.xfsc.dev/realms/master"},
		CredentialEndpoint:         "https://cloud-wallet.xfsc.dev/api/credential",
		BatchCredentialEndpoint:    strPtr("https://credential-issuer.eclipse.org/batch_credential"),
		DeferredCredentialEndpoint: strPtr("https://credential-issuer.eclipse.org/deferred_credential"),
		CredentialResponseEncryption: credential.CredentialRespEnc{
			AlgValuesSupported: []string{"ECDH-ES"},
			EncValuesSupported: []string{"A128GCM"},
			EncryptionRequired: false,
		},
		Display: []credential.LocalizedCredential{
			{Name: "Example Issuer", Locale: "en-US"},
			{Name: "Beispiel Issuer", Locale: "de-DE"},
		},
		CredentialConfigurationsSupported: map[string]credential.CredentialConfiguration{
			"DeveloperCredential": {
				Format:                               "jwt_vc_json",
				Scope:                                "Developer",
				CryptographicBindingMethodsSupported: []string{"did:example"},
				CredentialSigningAlgValuesSupported:  []string{"ES256"},
				CredentialDefinition: credential.CredentialDefinition{
					Type: []string{"VerifiableCredential", "DeveloperCredential"},
					CredentialSubject: map[string]credential.CredentialSubject{
						"given_name": {
							Display: []credential.Display{credential.Display{
								Name:   "Given Name",
								Locale: "en-US",
							}},
						},
						"family_name": {
							Display: []credential.Display{credential.Display{
								Name:   "Surname",
								Locale: "en-US",
							}},
						},
					},
				},
				ProofTypesSupported: map[string]credential.ProofType{
					"jwt": {
						ProofSigningAlgValuesSupported: []string{"ES256"},
					},
				},
				Display: []credential.LocalizedCredential{
					{
						Name:   "Developer Credential",
						Locale: "en-US",
						Logo: credential.DescriptiveURL{
							URL:             "https://www.eclipse.org/eclipse.org-common/themes/solstice/public/images/logo/eclipse-foundation-grey-orange.svg",
							AlternativeText: "Eclipse Foundation Logo",
						},
						BackgroundColor: "#FFFFFF",
						TextColor:       "#000000",
					},
				},
				Schema: map[string]interface{}{
					"$schema":     "https://json-schema.org/draft/2020-12/schema",
					"$id":         "https://example.com/developercredential.schema.json",
					"title":       "Developer Credential",
					"description": "A product from Acme's catalog",
					"type":        "object",
					"properties": map[string]interface{}{
						"given_name": map[string]interface{}{
							"description": "The unique identifier for a product",
							"type":        "string",
						},
						"family_name": map[string]interface{}{
							"description": "Name of the product",
							"type":        "string",
						},
					},
				},
				Subject: "issuer.tenant_space.DeveloperCredential",
			},
		},
	},
}
