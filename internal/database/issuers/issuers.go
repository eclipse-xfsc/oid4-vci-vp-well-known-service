package issuers

import (
	"context"
	"time"

	"github.com/eclipse-xfsc/oid4-vci-vp-library/model/credential"
)

type Store interface {
	Get(ctx context.Context, tenantID string) (*Issuer, error)
	Insert(ctx context.Context, issuer Issuer) error
	Update(ctx context.Context, tenantID, credentialIssuer string, update IssuerUpdate) error
	// List(ctx context.Context, tenantID string) ([]Issuer, error)
	// ListAll(ctx context.Context) ([]Issuer, error)
}

type Issuer struct {
	TenantID                       string
	CredentialIssuer               string
	AuthorizationServers           []string
	CredentialEndpoint             string
	BatchCredentialEndpoint        *string
	DeferredCredentialEndpoint     *string
	CredentialResponseEncryption   *CredentialRespEnc
	Display                        []credential.LocalizedCredential
	CredentialsSupported           []CredentialsSupported
	FirstSeen                      time.Time
	LastSeen                       time.Time
	SignedMetadata                 *string
	NotificationEndpoint           *string
	CredentialIdentifiersSupported bool
}

type CredentialRespEnc struct {
	AlgValuesSupported []string `json:"alg_values_supported"`
	EncValuesSupported []string `json:"enc_values_supported"`
	EncryptionRequired bool     `json:"encryption_required"`
}

type Locale string

type IssuerUpdate struct {
	AuthorizationServers           []string
	CredentialEndpoint             *string
	BatchCredentialEndpoint        *string
	DeferredCredentialEndpoint     *string
	CredentialResponseEncryption   *CredentialRespEnc
	CredentialIdentifiersSupported bool
	Display                        []credential.LocalizedCredential
	CredentialsSupported           []CredentialsSupported
	LastSeen                       *time.Time
	SignedMetadata                 *string
	NotificationEndpoint           *string
}

type CredentialsSupported struct {
	CredentialConfigurationID              string
	Format                                 string
	Scope                                  string
	CryptographicBindingMethodsSupported   []string
	CryptographicSigningAlgValuesSupported []string
	CredentialDefinition                   credential.CredentialDefinition
	ProofTypesSupported                    ProofTypesSupported
	Display                                []credential.LocalizedCredential
	Schema                                 map[string]interface{}
	Subject                                string
	Vct                                    *string
	Claims                                 map[string]interface{}
	Order                                  []string
	FirstSeen                              time.Time
	LastSeen                               time.Time
}

type CredentialSupportedRow struct {
	CredentialConfigurationID              *string
	Format                                 *string
	Scope                                  *string
	CryptographicBindingMethodsSupported   []string
	CryptographicSigningAlgValuesSupported []string
	CredentialDefinition                   *credential.CredentialDefinition
	ProofTypesSupported                    ProofTypesSupported
	Display                                []credential.LocalizedCredential
	Schema                                 map[string]interface{}
	Subject                                *string
	Vct                                    *string
	Claims                                 map[string]interface{}
	Order                                  []string
	FirstSeen                              time.Time
	LastSeen                               time.Time
}

type ProofTypesSupported map[string]credential.ProofType

type DescriptiveURL struct {
	URL             string `json:"url"`
	AlternativeText string `json:"alternative_text"`
}
