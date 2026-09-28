package service

import (
	"context"
	"errors"
	"time"

	ctxPkg "github.com/eclipse-xfsc/microservice-core-go/pkg/ctx"
	"github.com/eclipse-xfsc/oid4-vci-vp-library/model/credential"

	"github.com/eclipse-xfsc/oid4-vci-vp-well-known-service/internal/database"
	"github.com/eclipse-xfsc/oid4-vci-vp-well-known-service/internal/database/issuers"
)

type IssuerService struct {
	store issuers.Store
}

func NewIssuerService(store issuers.Store) IssuerService {
	return IssuerService{
		store: store,
	}
}

func (s IssuerService) GetIssuer(
	ctx context.Context,
	tenantID string,
	withInternal bool,
) (*credential.IssuerMetadata, error) {
	log := ctxPkg.GetLogger(ctx)

	issuer, err := s.store.GetIssuerRecord(ctx, tenantID)
	if err != nil {
		log.Error(err, "Issuer Record not found", nil)
		return nil, err
	}

	configurations := make(map[string]credential.CredentialConfiguration)

	for _, supported := range issuer.CredentialsSupported {
		id := supported.CredentialConfigurationID

		config := credential.CredentialConfiguration{
			Format: supported.Format,
			Scope:  supported.Scope,

			CryptographicBindingMethodsSupported: supported.CryptographicBindingMethodsSupported,
			CredentialSigningAlgValuesSupported:  supported.CryptographicSigningAlgValuesSupported,

			CredentialDefinition: &supported.CredentialDefinition,
			ProofTypesSupported:  supported.ProofTypesSupported,

			CredentialMetadata: supported.CredentialMetadata,

			Vct: supported.Vct,
		}

		if withInternal {
			config.Schema = supported.Schema
			config.Subject = supported.Subject
		}

		configurations[id] = config
	}

	metadata := &credential.IssuerMetadata{
		CredentialIssuer:     issuer.CredentialIssuer,
		AuthorizationServers: issuer.AuthorizationServers,
		CredentialEndpoint:   issuer.CredentialEndpoint,

		NonceEndpoint:              issuer.NonceEndpoint,
		DeferredCredentialEndpoint: issuer.DeferredCredentialEndpoint,
		NotificationEndpoint:       issuer.NotificationEndpoint,

		Display:        issuer.Display,
		SignedMetadata: issuer.SignedMetadata,

		CredentialConfigurationsSupported: configurations,
	}

	if issuer.CredentialResponseEncryption != nil {
		metadata.CredentialResponseEncryption = &credential.CredentialResponseEncryption{
			AlgValuesSupported: issuer.CredentialResponseEncryption.AlgValuesSupported,
			EncValuesSupported: issuer.CredentialResponseEncryption.EncValuesSupported,
			EncryptionRequired: issuer.CredentialResponseEncryption.EncryptionRequired,
		}
	}

	return metadata, nil
}

// UpsertIssuer stores the given issuer or updates the existing record.
func (s IssuerService) UpsertIssuer(
	ctx context.Context,
	tenantID string,
	issuerMetadata credential.IssuerMetadata,
) error {
	log := ctxPkg.GetLogger(ctx)

	storedIssuer, err := s.store.GetIssuerRecord(ctx, tenantID)
	if err != nil && !errors.Is(err, database.ErrNotFound) {
		return err
	}

	now := time.Now()

	configurations := make(
		[]issuers.CredentialsSupported,
		0,
		len(issuerMetadata.CredentialConfigurationsSupported),
	)

	for configurationID, supported := range issuerMetadata.CredentialConfigurationsSupported {
		var credentialDefinition credential.CredentialDefinition

		if supported.CredentialDefinition != nil {
			credentialDefinition = *supported.CredentialDefinition
		}

		configurations = append(
			configurations,
			issuers.CredentialsSupported{
				TenantID:                  tenantID,
				CredentialConfigurationID: configurationID,

				Format: supported.Format,
				Scope:  supported.Scope,

				CryptographicBindingMethodsSupported:   supported.CryptographicBindingMethodsSupported,
				CryptographicSigningAlgValuesSupported: supported.CredentialSigningAlgValuesSupported,

				CredentialDefinition: credentialDefinition,
				ProofTypesSupported:  supported.ProofTypesSupported,

				CredentialMetadata: supported.CredentialMetadata,

				Vct: supported.Vct,

				Schema:  supported.Schema,
				Subject: supported.Subject,

				FirstSeen: now,
				LastSeen:  now,
			},
		)
	}

	if storedIssuer == nil {
		newIssuer := issuers.Issuer{
			TenantID: tenantID,

			CredentialIssuer: issuerMetadata.CredentialIssuer,

			AuthorizationServers: issuerMetadata.AuthorizationServers,
			CredentialEndpoint:   issuerMetadata.CredentialEndpoint,

			NonceEndpoint:              issuerMetadata.NonceEndpoint,
			DeferredCredentialEndpoint: issuerMetadata.DeferredCredentialEndpoint,
			NotificationEndpoint:       issuerMetadata.NotificationEndpoint,

			CredentialsSupported: configurations,

			Display:        issuerMetadata.Display,
			SignedMetadata: issuerMetadata.SignedMetadata,

			FirstSeen: now,
			LastSeen:  now,
		}

		if issuerMetadata.CredentialResponseEncryption != nil {
			newIssuer.CredentialResponseEncryption = &issuers.CredentialRespEnc{
				AlgValuesSupported: issuerMetadata.CredentialResponseEncryption.AlgValuesSupported,
				EncValuesSupported: issuerMetadata.CredentialResponseEncryption.EncValuesSupported,
				EncryptionRequired: issuerMetadata.CredentialResponseEncryption.EncryptionRequired,
			}
		}

		if err := s.store.InsertIssuerRecord(ctx, newIssuer); err != nil {
			log.Error(
				err,
				"failed to insert new issuer",
				"prev",
				errors.Unwrap(err),
			)

			return err
		}

		return nil
	}

	update := issuers.IssuerUpdate{
		AuthorizationServers: issuerMetadata.AuthorizationServers,
		CredentialEndpoint:   &issuerMetadata.CredentialEndpoint,

		NonceEndpoint:              issuerMetadata.NonceEndpoint,
		DeferredCredentialEndpoint: issuerMetadata.DeferredCredentialEndpoint,
		NotificationEndpoint:       issuerMetadata.NotificationEndpoint,

		SignedMetadata: issuerMetadata.SignedMetadata,
		Display:        issuerMetadata.Display,

		LastSeen: &now,
	}

	if issuerMetadata.CredentialResponseEncryption != nil {
		update.CredentialResponseEncryption = &issuers.CredentialRespEnc{
			AlgValuesSupported: issuerMetadata.CredentialResponseEncryption.AlgValuesSupported,
			EncValuesSupported: issuerMetadata.CredentialResponseEncryption.EncValuesSupported,
			EncryptionRequired: issuerMetadata.CredentialResponseEncryption.EncryptionRequired,
		}
	}

	existingConfigurations := make(
		map[string]issuers.CredentialsSupported,
		len(storedIssuer.CredentialsSupported),
	)

	for _, existing := range storedIssuer.CredentialsSupported {
		existingConfigurations[existing.CredentialConfigurationID] = existing
	}

	finalConfigurations := make(
		[]issuers.CredentialsSupported,
		0,
		len(configurations),
	)

	for _, configuration := range configurations {
		if existing, ok := existingConfigurations[configuration.CredentialConfigurationID]; ok {
			configuration.FirstSeen = existing.FirstSeen
		}

		configuration.LastSeen = now

		finalConfigurations = append(
			finalConfigurations,
			configuration,
		)
	}

	update.CredentialsSupported = finalConfigurations

	if err := s.store.UpdateIssuerRecord(
		ctx,
		tenantID,
		issuerMetadata.CredentialIssuer,
		update,
	); err != nil {
		log.Error(
			err,
			"failed to update existing issuer",
		)

		return err
	}

	return nil
}

// UpsertConfiguration stores or updates a single credential configuration.
func (s IssuerService) UpsertConfiguration(
	ctx context.Context,
	tenantID string,
	configurationID string,
	configuration credential.CredentialConfiguration,
) error {
	log := ctxPkg.GetLogger(ctx)

	storedConfigurations, err := s.store.GetConfigurationsRecord(
		ctx,
		tenantID,
	)
	if err != nil && !errors.Is(err, database.ErrNotFound) {
		return err
	}

	now := time.Now()

	var credentialDefinition credential.CredentialDefinition

	if configuration.CredentialDefinition != nil {
		credentialDefinition = *configuration.CredentialDefinition
	}

	supported := issuers.CredentialsSupported{
		TenantID:                  tenantID,
		CredentialConfigurationID: configurationID,

		Format: configuration.Format,
		Scope:  configuration.Scope,

		CryptographicBindingMethodsSupported:   configuration.CryptographicBindingMethodsSupported,
		CryptographicSigningAlgValuesSupported: configuration.CredentialSigningAlgValuesSupported,

		CredentialDefinition: credentialDefinition,
		ProofTypesSupported:  configuration.ProofTypesSupported,

		CredentialMetadata: configuration.CredentialMetadata,

		Vct: configuration.Vct,

		Schema:  configuration.Schema,
		Subject: configuration.Subject,

		LastSeen: now,
	}

	isNew := true

	finalConfigurations := make(
		[]issuers.CredentialsSupported,
		0,
		len(storedConfigurations)+1,
	)

	for _, existing := range storedConfigurations {
		if existing.CredentialConfigurationID == configurationID {
			isNew = false
			supported.FirstSeen = existing.FirstSeen

			continue
		}

		finalConfigurations = append(
			finalConfigurations,
			existing,
		)
	}

	if isNew {
		supported.FirstSeen = now
	}

	finalConfigurations = append(
		finalConfigurations,
		supported,
	)

	if err := s.store.UpdateConfigurationsSupported(
		ctx,
		tenantID,
		finalConfigurations,
	); err != nil {
		log.Error(
			err,
			"failed to update credential configuration",
		)

		return err
	}

	return nil
}
