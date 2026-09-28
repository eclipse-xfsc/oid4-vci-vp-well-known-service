package issuers

import (
	"context"
	"time"

	"github.com/eclipse-xfsc/oid4-vci-vp-library/model/credential"
)

type Store interface {
	GetIssuerRecord(
		ctx context.Context,
		tenantID string,
	) (*Issuer, error)

	GetConfigurationsRecord(
		ctx context.Context,
		tenantID string,
	) ([]CredentialsSupported, error)

	InsertIssuerRecord(
		ctx context.Context,
		issuer Issuer,
	) error

	UpdateIssuerRecord(
		ctx context.Context,
		tenantID string,
		credentialIssuer string,
		update IssuerUpdate,
	) error

	InsertConfigurationsSupported(
		ctx context.Context,
		tenantID string,
		cs []CredentialsSupported,
	) error

	UpdateConfigurationsSupported(
		ctx context.Context,
		tenantID string,
		update []CredentialsSupported,
	) error
}

type Issuer struct {
	TenantID             string
	CredentialIssuer     string
	AuthorizationServers []string
	CredentialEndpoint   string

	NonceEndpoint              *string
	DeferredCredentialEndpoint *string
	NotificationEndpoint       *string

	CredentialResponseEncryption *CredentialRespEnc

	Display        []credential.LocalizedCredential
	SignedMetadata *string

	CredentialsSupported []CredentialsSupported

	FirstSeen time.Time
	LastSeen  time.Time
}

type CredentialRespEnc struct {
	AlgValuesSupported []string `json:"alg_values_supported"`
	EncValuesSupported []string `json:"enc_values_supported"`
	EncryptionRequired bool     `json:"encryption_required"`
}

type IssuerUpdate struct {
	AuthorizationServers []string
	CredentialEndpoint   *string

	NonceEndpoint              *string
	DeferredCredentialEndpoint *string
	NotificationEndpoint       *string

	CredentialResponseEncryption *CredentialRespEnc

	Display        []credential.LocalizedCredential
	SignedMetadata *string

	CredentialsSupported []CredentialsSupported

	LastSeen *time.Time
}

type CredentialsSupported struct {
	TenantID                  string
	CredentialConfigurationID string

	Format string
	Scope  string

	CryptographicBindingMethodsSupported   []string
	CryptographicSigningAlgValuesSupported []string

	CredentialDefinition credential.CredentialDefinition
	ProofTypesSupported  ProofTypesSupported

	CredentialMetadata *credential.CredentialMetadata

	Vct *string

	Schema  map[string]interface{}
	Subject string

	FirstSeen time.Time
	LastSeen  time.Time
}

type CredentialSupportedRow struct {
	TenantID                  string
	CredentialConfigurationID *string

	Format *string
	Scope  *string

	CryptographicBindingMethodsSupported   []string
	CryptographicSigningAlgValuesSupported []string

	CredentialDefinition *credential.CredentialDefinition
	ProofTypesSupported  ProofTypesSupported

	CredentialMetadata *credential.CredentialMetadata

	Vct *string

	Schema  map[string]interface{}
	Subject *string

	FirstSeen time.Time
	LastSeen  time.Time
}

type ProofTypesSupported map[credential.ProofVariant]credential.ProofType
