package service

import (
	"context"
	"errors"
	"time"

	"github.com/eclipse-xfsc/oid4-vci-vp-library/model/credential"

	ctxPkg "github.com/eclipse-xfsc/microservice-core-go/pkg/ctx"

	"github.com/eclipse-xfsc/oid4-vci-vp-well-known-service/internal/database"

	"github.com/eclipse-xfsc/oid4-vci-vp-well-known-service/internal/database/issuers"
)

type IssuerService struct {
	store issuers.Store
}

func NewIssuerService(store issuers.Store) IssuerService {
	return IssuerService{store: store}
}

func (s IssuerService) GetIssuer(ctx context.Context, tenantID string, withInternal bool) (*credential.IssuerMetadata, error) {
	log := ctxPkg.GetLogger(ctx)
	issuer, err := s.store.GetIssuerRecord(ctx, tenantID)
	if err != nil {
		log.Error(err, "Issuer Record not found", nil)
		return nil, err
	}

	cs := make(map[string]credential.CredentialConfiguration)
	for _, supported := range issuer.CredentialsSupported {
		id := supported.CredentialConfigurationID

		cs[id] = credential.CredentialConfiguration{
			Format:                               supported.Format,
			Scope:                                supported.Scope,
			CryptographicBindingMethodsSupported: supported.CryptographicBindingMethodsSupported,
			CredentialSigningAlgValuesSupported:  supported.CryptographicSigningAlgValuesSupported,
			ProofTypesSupported:                  supported.ProofTypesSupported,
			CredentialDefinition:                 supported.CredentialDefinition,
			Display:                              supported.Display,
			Vct:                                  supported.Vct,
			Claims:                               supported.Claims,
			Order:                                supported.Order,
			Schema:                               supported.Schema,
			Subject:                              supported.Subject,
		}

		if withInternal {
			config := cs[id]
			config.Schema = supported.Schema
			config.Subject = supported.Subject
			cs[id] = config
		}
	}

	iss := &credential.IssuerMetadata{
		CredentialIssuer:                  issuer.CredentialIssuer,
		CredentialEndpoint:                issuer.CredentialEndpoint,
		AuthorizationServers:              issuer.AuthorizationServers,
		BatchCredentialEndpoint:           issuer.BatchCredentialEndpoint,
		DeferredCredentialEndpoint:        issuer.DeferredCredentialEndpoint,
		NotificationEndpoint:              issuer.NotificationEndpoint,
		Display:                           issuer.Display,
		CredentialIdentifiersSupported:    issuer.CredentialIdentifiersSupported,
		SignedMetadata:                    issuer.SignedMetadata,
		CredentialConfigurationsSupported: cs,
		CredentialResponseEncryption:      credential.CredentialRespEnc(*issuer.CredentialResponseEncryption),
	}

	if issuer.CredentialResponseEncryption != nil {
		iss.CredentialResponseEncryption = credential.CredentialRespEnc{
			AlgValuesSupported: issuer.CredentialResponseEncryption.AlgValuesSupported,
			EncValuesSupported: issuer.CredentialResponseEncryption.EncValuesSupported,
			EncryptionRequired: issuer.CredentialResponseEncryption.EncryptionRequired,
		}
	}

	return iss, nil
}

// UpsertIssuer will store the given issuer or, if it already exists, update the existing record
func (s IssuerService) UpsertIssuer(ctx context.Context, tenantID string, issuer credential.IssuerMetadata) error {
	log := ctxPkg.GetLogger(ctx)

	storedIssuer, err := s.store.GetIssuerRecord(ctx, tenantID)
	if err != nil && !errors.Is(err, database.ErrNotFound) {
		return err
	}
	now := time.Now()
	cs := make([]issuers.CredentialsSupported, 0)
	for ccid, supported := range issuer.CredentialConfigurationsSupported {

		sup := issuers.CredentialsSupported{
			CredentialConfigurationID:              ccid,
			Format:                                 supported.Format,
			Scope:                                  supported.Scope,
			CryptographicBindingMethodsSupported:   supported.CryptographicBindingMethodsSupported,
			CryptographicSigningAlgValuesSupported: supported.CredentialSigningAlgValuesSupported,
			CredentialDefinition:                   supported.CredentialDefinition,
			ProofTypesSupported:                    supported.ProofTypesSupported,
			Schema:                                 supported.Schema,
			Subject:                                supported.Subject,
			Display:                                supported.Display,
			Vct:                                    supported.Vct,
			Claims:                                 supported.Claims,
			Order:                                  supported.Order,
			LastSeen:                               now,
			FirstSeen:                              now,
		}
		cs = append(cs, sup)
	}

	isNew := storedIssuer == nil

	if isNew {
		storedIssuer = &issuers.Issuer{
			TenantID:                   tenantID,
			CredentialIssuer:           issuer.CredentialIssuer,
			AuthorizationServers:       issuer.AuthorizationServers,
			CredentialEndpoint:         issuer.CredentialEndpoint,
			BatchCredentialEndpoint:    issuer.BatchCredentialEndpoint,
			DeferredCredentialEndpoint: issuer.DeferredCredentialEndpoint,
			NotificationEndpoint:       issuer.NotificationEndpoint,
			CredentialResponseEncryption: &issuers.CredentialRespEnc{
				AlgValuesSupported: issuer.CredentialResponseEncryption.AlgValuesSupported,
				EncValuesSupported: issuer.CredentialResponseEncryption.EncValuesSupported,
				EncryptionRequired: issuer.CredentialResponseEncryption.EncryptionRequired,
			},
			CredentialsSupported:           cs,
			LastSeen:                       now,
			FirstSeen:                      now,
			Display:                        issuer.Display,
			SignedMetadata:                 issuer.SignedMetadata,
			CredentialIdentifiersSupported: issuer.CredentialIdentifiersSupported,
		}

		if err := s.store.InsertIssuerRecord(ctx, *storedIssuer); err != nil {
			log.Error(err, "failed to insert new issuer", "prev", errors.Unwrap(err))
			return err
		}

		return nil
	}

	update := issuers.IssuerUpdate{
		AuthorizationServers:           issuer.AuthorizationServers,
		CredentialEndpoint:             &issuer.CredentialEndpoint,
		BatchCredentialEndpoint:        issuer.BatchCredentialEndpoint,
		DeferredCredentialEndpoint:     issuer.DeferredCredentialEndpoint,
		NotificationEndpoint:           issuer.NotificationEndpoint,
		SignedMetadata:                 issuer.SignedMetadata,
		CredentialIdentifiersSupported: issuer.CredentialIdentifiersSupported,
		Display:                        issuer.Display,
		LastSeen:                       &now,
	}

	finalCs := make([]issuers.CredentialsSupported, 0)

	//Update last seen if exist
	for i, c := range finalCs {
		for _, x := range cs {
			if x.CredentialConfigurationID == c.CredentialConfigurationID {
				finalCs[i].LastSeen = now
				continue
			}
		}
	}

	for _, x := range cs {
		found := false
		for _, c := range finalCs {
			if x.CredentialConfigurationID == c.CredentialConfigurationID {
				found = true
				break
			}
		}

		if !found {
			finalCs = append(finalCs, x)
		}
	}

	update.CredentialsSupported = finalCs

	if err := s.store.UpdateIssuerRecord(ctx, tenantID, issuer.CredentialIssuer, update); err != nil {
		log.Error(err, "failed to update existing issuer")
		return err
	}

	return nil
}

// UpsertIssuer will store the given issuer or, if it already exists, update the existing record
func (s IssuerService) UpsertConfiguration(ctx context.Context, tenantID string, configurationId string, configuration credential.CredentialConfiguration) error {
	log := ctxPkg.GetLogger(ctx)

	storedConfiguration, err := s.store.GetConfigurationsRecord(ctx, tenantID)
	if err != nil && !errors.Is(err, database.ErrNotFound) {
		return err
	}
	now := time.Now()

	sup := issuers.CredentialsSupported{
		TenantID:                               tenantID,
		CredentialConfigurationID:              configurationId,
		Format:                                 configuration.Format,
		Scope:                                  configuration.Scope,
		CryptographicBindingMethodsSupported:   configuration.CryptographicBindingMethodsSupported,
		CryptographicSigningAlgValuesSupported: configuration.CredentialSigningAlgValuesSupported,
		CredentialDefinition:                   configuration.CredentialDefinition,
		ProofTypesSupported:                    configuration.ProofTypesSupported,
		Schema:                                 configuration.Schema,
		Subject:                                configuration.Subject,
		Display:                                configuration.Display,
		Vct:                                    configuration.Vct,
		Claims:                                 configuration.Claims,
		Order:                                  configuration.Order,
		LastSeen:                               now,
	}
	isNew := true
	finalCs := make([]issuers.CredentialsSupported, 0)
	for _, c := range storedConfiguration {
		if c.CredentialConfigurationID == configurationId {
			isNew = false
			continue
		}
		finalCs = append(finalCs, c)
	}
	// cs := make([]issuers.CredentialsSupported, 0)

	if isNew {
		sup.FirstSeen = now
	}

	finalCs = append(finalCs, sup)
	if err := s.store.UpdateConfigurationsSupported(ctx, tenantID, finalCs); err != nil {
		log.Error(err, "failed to update existing issuer")
		return err
	}

	return nil
}
