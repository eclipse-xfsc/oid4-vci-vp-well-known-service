package postgres

import (
	"context"
	"fmt"

	"github.com/eclipse-xfsc/microservice-core-go/pkg/logr"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/eclipse-xfsc/oid4-vci-vp-well-known-service/internal/database"
	"github.com/eclipse-xfsc/oid4-vci-vp-well-known-service/internal/database/issuers"
	"github.com/eclipse-xfsc/oid4-vci-vp-well-known-service/internal/database/postgres"
)

type Store struct {
	log logr.Logger
	db  *pgxpool.Pool
	sq  squirrel.StatementBuilderType
}

var _ issuers.Store = Store{}

const (
	colTenantId                       = "tenant_id"
	colCredentialIssuer               = "credential_issuer"
	colAuthorizationServers           = "authorization_servers"
	colCredentialEndpoint             = "credential_endpoint"
	colBatchCredentialEndpoint        = "batch_credential_endpoint"
	colDeferredCredentialEndpoint     = "deferred_credential_endpoint"
	colCredentialResponseEncryption   = "credential_response_encryption"
	colDisplay                        = "display"
	colFirstSeen                      = "first_seen"
	colLastSeen                       = "last_seen"
	colSignedMetaData                 = "signed_metadata"
	colNotificationEndpoint           = "notification_endpoint"
	colCredentialIdentifiersSupported = "credential_identifiers_supported"

	colCredentialConfigurationID            = "credential_configuration_id"
	colFormat                               = "format"
	colScope                                = "scope"
	colCryptographicBindingMethodsSupported = "cryptographic_binding_methods_supported"
	colSigningAlgValuesSupported            = "credential_signing_alg_values_supported"
	colCredentialDefinition                 = "credential_definition"
	colProofTypesSupported                  = "proof_types_supported"
	colSchema                               = "schema"
	colSubject                              = "subject"
	colVct                                  = "vct"
	colClaims                               = "claims"
	colOrder                                = "\"order\""
)

func NewStore(db *pgxpool.Pool, logger logr.Logger) Store {
	return Store{
		log: logger,
		db:  db,
		sq:  postgres.StmtBuilderDollar(),
	}
}
func (s Store) Get(ctx context.Context, tenantID string) (*issuers.Issuer, error) {
	rows, err := s.list(
		ctx,
		colTenantId,
		squirrel.Eq{postgres.Prepend(postgres.TblIssuers, colTenantId): tenantID},
	)
	if err != nil {
		return nil, err
	}

	if len(rows) < 1 {
		return nil, database.ErrNotFound
	}

	return &rows[0], nil
}

func (s Store) Insert(ctx context.Context, issuer issuers.Issuer) error {
	query := s.sq.
		Insert(postgres.TblIssuers).
		Columns(
			colTenantId, colCredentialIssuer,
			colAuthorizationServers, colCredentialEndpoint,
			colBatchCredentialEndpoint, colDeferredCredentialEndpoint,
			colCredentialResponseEncryption, colDisplay,
			colFirstSeen, colLastSeen, colSignedMetaData,
			colNotificationEndpoint, colCredentialIdentifiersSupported,
		).
		Values(
			issuer.TenantID, issuer.CredentialIssuer,
			issuer.AuthorizationServers, issuer.CredentialEndpoint,
			issuer.BatchCredentialEndpoint, issuer.DeferredCredentialEndpoint,
			issuer.CredentialResponseEncryption,
			issuer.Display, issuer.FirstSeen, issuer.LastSeen,
			issuer.SignedMetadata, issuer.NotificationEndpoint, issuer.CredentialIdentifiersSupported,
		)

	sql, params, err := query.ToSql()
	if err != nil {
		return database.NewError("failed to build query", err)
	}

	if _, err := s.db.Exec(ctx, sql, params...); err != nil {
		return database.NewError("failed to execute query", err)
	}

	return s.insertCredentialsSupported(ctx, issuer.TenantID, issuer.CredentialsSupported)
}

func (s Store) insertCredentialsSupported(ctx context.Context, tenantID string, cs []issuers.CredentialsSupported) error {
	query := s.sq.
		Insert(postgres.TblCredentialsSupported).
		Columns(
			colTenantId, colCredentialConfigurationID, colFormat, colScope,
			colCryptographicBindingMethodsSupported, colSigningAlgValuesSupported,
			colCredentialDefinition, colProofTypesSupported, colSchema, colSubject,
			colFirstSeen, colLastSeen, colDisplay, colVct, colClaims, colOrder,
		)

	for _, supported := range cs {
		query = query.Values(
			tenantID, supported.CredentialConfigurationID, supported.Format, supported.Scope,
			supported.CryptographicBindingMethodsSupported, supported.CryptographicSigningAlgValuesSupported,
			supported.CredentialDefinition, supported.ProofTypesSupported, supported.Schema, supported.Subject,
			supported.FirstSeen, supported.LastSeen, supported.Display, supported.Vct, supported.Claims, supported.Order,
		)
	}

	sql, params, err := query.ToSql()
	if err != nil {
		return database.NewError("failed to build query", err)
	}

	if _, err := s.db.Exec(ctx, sql, params...); err != nil {
		s.log.Error(err, "failed to insert credentials supported")
		return database.NewError("failed to insert credentials supported", err)
	}

	return nil
}

func (s Store) Update(ctx context.Context, tenantID, issuer string, update issuers.IssuerUpdate) error {
	query := s.sq.
		Update(postgres.TblIssuers).
		Where(squirrel.Eq{colCredentialIssuer: issuer}).
		Where(squirrel.Eq{colTenantId: tenantID})

	if update.CredentialEndpoint != nil {
		query = query.Set(colCredentialEndpoint, update.CredentialEndpoint)
	}

	if update.AuthorizationServers != nil {
		query = query.Set(colAuthorizationServers, update.AuthorizationServers)
	}

	if update.LastSeen != nil {
		query = query.Set(colLastSeen, update.LastSeen)
	}

	sql, params, err := query.ToSql()
	if err != nil {
		return database.NewError("failed to build query", err)
	}

	if _, err := s.db.Exec(ctx, sql, params...); err != nil {
		return database.NewError("failed to execute query", err)
	}

	return s.updateCredentialsSupported(ctx, tenantID, update)
}

func (s Store) updateCredentialsSupported(ctx context.Context, tenantID string, update issuers.IssuerUpdate) error {
	query := s.sq.Delete(postgres.TblCredentialsSupported).Where(squirrel.Eq{colTenantId: tenantID})

	sql, params, err := query.ToSql()
	if err != nil {
		return database.NewError("failed to build query", err)
	}

	if _, err := s.db.Exec(ctx, sql, params...); err != nil {
		return database.NewError("failed to update credentials supported", err)
	}

	return s.insertCredentialsSupported(ctx, tenantID, update.CredentialsSupported)
}

func (s Store) list(ctx context.Context, orderBy string, where ...any) ([]issuers.Issuer, error) {
	columns := postgres.PrependAll(postgres.TblIssuers,
		colTenantId, colCredentialIssuer,
		colAuthorizationServers, colCredentialEndpoint,
		colBatchCredentialEndpoint, colDeferredCredentialEndpoint,
		colCredentialResponseEncryption, colDisplay,
		colFirstSeen, colLastSeen, colSignedMetaData,
		colNotificationEndpoint, colCredentialIdentifiersSupported,
	)

	columns = append(columns, postgres.PrependAll(postgres.TblCredentialsSupported,
		colCredentialConfigurationID, colFormat, colScope,
		colCryptographicBindingMethodsSupported, colSigningAlgValuesSupported,
		colCredentialDefinition, colProofTypesSupported, colSchema, colSubject, colDisplay, colVct,
		colClaims, colOrder, colFirstSeen, colLastSeen,
	)...)

	query := s.sq.
		Select(columns...).
		From(postgres.TblIssuers).
		LeftJoin(fmt.Sprintf(
			"%s ON %s.%s=%s.%s",
			postgres.TblCredentialsSupported,
			postgres.TblIssuers, colTenantId,
			postgres.TblCredentialsSupported, colTenantId,
		)).
		OrderBy(postgres.Prepend(postgres.TblIssuers, colCredentialIssuer)).
		OrderBy(postgres.Prepend(postgres.TblIssuers, orderBy))

	for _, wh := range where {
		query = query.Where(wh)
	}

	sql, params, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := s.db.Query(ctx, sql, params...)
	if err != nil {
		return nil, err
	}

	var out []issuers.Issuer
	var previous *issuers.Issuer
	for rows.Next() {
		var issuer issuers.Issuer
		var csr issuers.CredentialSupportedRow

		err := rows.Scan(
			&issuer.TenantID, &issuer.CredentialIssuer,
			&issuer.AuthorizationServers, &issuer.CredentialEndpoint,
			&issuer.BatchCredentialEndpoint, &issuer.DeferredCredentialEndpoint,
			&issuer.CredentialResponseEncryption, &issuer.Display,
			&issuer.FirstSeen, &issuer.LastSeen, &issuer.SignedMetadata, &issuer.NotificationEndpoint, &issuer.CredentialIdentifiersSupported,
			&csr.CredentialConfigurationID, &csr.Format, &csr.Scope,
			&csr.CryptographicBindingMethodsSupported, &csr.CryptographicSigningAlgValuesSupported,
			&csr.CredentialDefinition, &csr.ProofTypesSupported,
			&csr.Schema, &csr.Subject, &csr.Display, &csr.Vct, &csr.Claims, &csr.Order, &csr.FirstSeen, &csr.LastSeen,
		)
		if err != nil {
			s.log.Error(err, "failed to scan")
			return nil, err
		}

		// join can produce null values, if there is no matching row
		if csr.CredentialConfigurationID != nil {
			issuer.CredentialsSupported = []issuers.CredentialsSupported{{
				CredentialConfigurationID:              *csr.CredentialConfigurationID,
				Format:                                 *csr.Format,
				Scope:                                  *csr.Scope,
				CryptographicBindingMethodsSupported:   csr.CryptographicBindingMethodsSupported,
				CryptographicSigningAlgValuesSupported: csr.CryptographicSigningAlgValuesSupported,
				CredentialDefinition:                   *csr.CredentialDefinition,
				ProofTypesSupported:                    csr.ProofTypesSupported,
				Display:                                csr.Display,
				Schema:                                 csr.Schema,
				Subject:                                *csr.Subject,
				Vct:                                    csr.Vct,
				Claims:                                 csr.Claims,
				Order:                                  csr.Order,
				FirstSeen:                              csr.FirstSeen,
				LastSeen:                               csr.LastSeen,
			}}
		}

		// first row
		if previous == nil {
			previous = &issuer
			continue
		}

		// new issuer
		if previous.TenantID != issuer.TenantID {
			out = append(out, *previous)
			previous = &issuer
			continue
		}

		// same issuer as before, just append new csr
		previous.CredentialsSupported = append(previous.CredentialsSupported, issuer.CredentialsSupported[0])
	}

	if previous != nil {
		// always append the last issuer
		out = append(out, *previous)
	}

	return out, nil
}
