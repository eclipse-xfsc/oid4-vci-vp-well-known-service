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
	issuer, err := s.store.Get(ctx, tenantID)
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

	storedIssuer, err := s.store.Get(ctx, tenantID)
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

		if err := s.store.Insert(ctx, *storedIssuer); err != nil {
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

	//Kick out outdated
	for _, cs := range storedIssuer.CredentialsSupported {
		if cs.LastSeen.Add(time.Second * 40).Before(now) {
			continue
		}
		finalCs = append(finalCs, cs)
	}

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

	if err := s.store.Update(ctx, tenantID, issuer.CredentialIssuer, update); err != nil {
		log.Error(err, "failed to update existing issuer")
		return err
	}

	return nil
}
