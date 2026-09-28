package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/eclipse-xfsc/microservice-core-go/pkg/logr"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/eclipse-xfsc/oid4-vci-vp-well-known-service/config"
	"github.com/eclipse-xfsc/oid4-vci-vp-well-known-service/internal/database"
	"github.com/eclipse-xfsc/oid4-vci-vp-well-known-service/internal/database/issuers"
	"github.com/eclipse-xfsc/oid4-vci-vp-well-known-service/internal/database/postgres"
)

type Store struct {
	log logr.Logger
	db  *pgxpool.Pool
	sq  squirrel.StatementBuilderType
	cf  config.Config
}

var _ issuers.Store = Store{}

const (
	// issuers
	colTenantId                     = "tenant_id"
	colCredentialIssuer             = "credential_issuer"
	colAuthorizationServers         = "authorization_servers"
	colCredentialEndpoint           = "credential_endpoint"
	colNonceEndpoint                = "nonce_endpoint"
	colDeferredCredentialEndpoint   = "deferred_credential_endpoint"
	colNotificationEndpoint         = "notification_endpoint"
	colCredentialResponseEncryption = "credential_response_encryption"
	colDisplay                      = "display"
	colSignedMetaData               = "signed_metadata"
	colFirstSeen                    = "first_seen"
	colLastSeen                     = "last_seen"

	// credentials_supported
	colCredentialConfigurationID            = "credential_configuration_id"
	colFormat                               = "format"
	colScope                                = "scope"
	colCryptographicBindingMethodsSupported = "cryptographic_binding_methods_supported"
	colSigningAlgValuesSupported            = "credential_signing_alg_values_supported"
	colCredentialDefinition                 = "credential_definition"
	colProofTypesSupported                  = "proof_types_supported"
	colCredentialMetadata                   = "credential_metadata"
	colVct                                  = "vct"

	// internal XFSC fields
	colSchema  = "schema"
	colSubject = "subject"
)

func NewStore(
	db *pgxpool.Pool,
	logger logr.Logger,
	config config.Config,
) Store {
	return Store{
		log: logger,
		db:  db,
		sq:  postgres.StmtBuilderDollar(),
		cf:  config,
	}
}

func (s Store) GetIssuerRecord(
	ctx context.Context,
	tenantID string,
) (*issuers.Issuer, error) {
	rows, err := s.listIssuers(
		ctx,
		colTenantId,
		squirrel.Eq{
			postgres.Prepend(
				postgres.TblIssuers,
				colTenantId,
			): tenantID,
		},
	)
	if err != nil {
		return nil, err
	}

	if len(rows) < 1 {
		return nil, database.ErrNotFound
	}

	return &rows[0], nil
}

func (s Store) GetConfigurationsRecord(
	ctx context.Context,
	tenantID string,
) ([]issuers.CredentialsSupported, error) {
	rows, err := s.listCredentialConfigurations(
		ctx,
		colCredentialConfigurationID,
		squirrel.Eq{
			postgres.Prepend(
				postgres.TblCredentialsSupported,
				colTenantId,
			): tenantID,
		},
	)
	if err != nil {
		return nil, err
	}

	if len(rows) < 1 {
		return nil, database.ErrNotFound
	}

	return rows, nil
}

func (s Store) InsertIssuerRecord(
	ctx context.Context,
	issuer issuers.Issuer,
) error {
	query := s.sq.
		Insert(postgres.TblIssuers).
		Columns(
			colTenantId,
			colCredentialIssuer,
			colAuthorizationServers,
			colCredentialEndpoint,
			colNonceEndpoint,
			colDeferredCredentialEndpoint,
			colNotificationEndpoint,
			colCredentialResponseEncryption,
			colDisplay,
			colSignedMetaData,
			colFirstSeen,
			colLastSeen,
		).
		Values(
			issuer.TenantID,
			issuer.CredentialIssuer,
			issuer.AuthorizationServers,
			issuer.CredentialEndpoint,
			issuer.NonceEndpoint,
			issuer.DeferredCredentialEndpoint,
			issuer.NotificationEndpoint,
			issuer.CredentialResponseEncryption,
			issuer.Display,
			issuer.SignedMetadata,
			issuer.FirstSeen,
			issuer.LastSeen,
		)

	sql, params, err := query.ToSql()
	if err != nil {
		return database.NewError(
			"failed to build query",
			err,
		)
	}

	if _, err := s.db.Exec(
		ctx,
		sql,
		params...,
	); err != nil {
		return database.NewError(
			"failed to execute query",
			err,
		)
	}

	if len(issuer.CredentialsSupported) == 0 {
		return nil
	}

	return s.InsertConfigurationsSupported(
		ctx,
		issuer.TenantID,
		issuer.CredentialsSupported,
	)
}

func (s Store) InsertConfigurationsSupported(
	ctx context.Context,
	tenantID string,
	cs []issuers.CredentialsSupported,
) error {
	if len(cs) == 0 {
		return nil
	}

	query := s.sq.
		Insert(postgres.TblCredentialsSupported).
		Columns(
			colTenantId,
			colCredentialConfigurationID,
			colFormat,
			colScope,
			colCryptographicBindingMethodsSupported,
			colSigningAlgValuesSupported,
			colCredentialDefinition,
			colProofTypesSupported,
			colCredentialMetadata,
			colVct,
			colSchema,
			colSubject,
			colFirstSeen,
			colLastSeen,
		)

	for _, supported := range cs {
		query = query.Values(
			tenantID,
			supported.CredentialConfigurationID,
			supported.Format,
			supported.Scope,
			supported.CryptographicBindingMethodsSupported,
			supported.CryptographicSigningAlgValuesSupported,
			supported.CredentialDefinition,
			supported.ProofTypesSupported,
			supported.CredentialMetadata,
			supported.Vct,
			supported.Schema,
			supported.Subject,
			supported.FirstSeen,
			supported.LastSeen,
		)
	}

	sql, params, err := query.ToSql()
	if err != nil {
		return database.NewError(
			"failed to build query",
			err,
		)
	}

	if _, err := s.db.Exec(
		ctx,
		sql,
		params...,
	); err != nil {
		s.log.Error(
			err,
			"failed to insert credentials supported",
		)

		return database.NewError(
			"failed to insert credentials supported",
			err,
		)
	}

	return nil
}

func (s Store) UpdateIssuerRecord(
	ctx context.Context,
	tenantID string,
	issuer string,
	update issuers.IssuerUpdate,
) error {
	query := s.sq.
		Update(postgres.TblIssuers).
		Where(
			squirrel.Eq{
				colCredentialIssuer: issuer,
			},
		).
		Where(
			squirrel.Eq{
				colTenantId: tenantID,
			},
		)

	if update.CredentialEndpoint != nil {
		query = query.Set(
			colCredentialEndpoint,
			*update.CredentialEndpoint,
		)
	}

	if update.AuthorizationServers != nil {
		query = query.Set(
			colAuthorizationServers,
			update.AuthorizationServers,
		)
	}

	if update.NonceEndpoint != nil {
		query = query.Set(
			colNonceEndpoint,
			update.NonceEndpoint,
		)
	}

	if update.DeferredCredentialEndpoint != nil {
		query = query.Set(
			colDeferredCredentialEndpoint,
			update.DeferredCredentialEndpoint,
		)
	}

	if update.NotificationEndpoint != nil {
		query = query.Set(
			colNotificationEndpoint,
			update.NotificationEndpoint,
		)
	}

	if update.CredentialResponseEncryption != nil {
		query = query.Set(
			colCredentialResponseEncryption,
			update.CredentialResponseEncryption,
		)
	}

	if update.Display != nil {
		query = query.Set(
			colDisplay,
			update.Display,
		)
	}

	if update.SignedMetadata != nil {
		query = query.Set(
			colSignedMetaData,
			update.SignedMetadata,
		)
	}

	if update.LastSeen != nil {
		query = query.Set(
			colLastSeen,
			*update.LastSeen,
		)
	}

	sql, params, err := query.ToSql()
	if err != nil {
		return database.NewError(
			"failed to build query",
			err,
		)
	}

	if _, err := s.db.Exec(
		ctx,
		sql,
		params...,
	); err != nil {
		return database.NewError(
			"failed to execute query",
			err,
		)
	}

	return s.UpdateConfigurationsSupported(
		ctx,
		tenantID,
		update.CredentialsSupported,
	)
}

func (s Store) UpdateConfigurationsSupported(
	ctx context.Context,
	tenantID string,
	update []issuers.CredentialsSupported,
) error {
	if len(update) == 0 {
		return nil
	}

	now := time.Now()

	ids := make(
		[]string,
		0,
		len(update),
	)

	active := make(
		[]issuers.CredentialsSupported,
		0,
		len(update),
	)

	for _, configuration := range update {
		ids = append(
			ids,
			configuration.CredentialConfigurationID,
		)

		expiration := configuration.LastSeen.Add(
			time.Second *
				time.Duration(
					s.cf.CredentialConfigurationExpiration,
				),
		)

		if expiration.Before(now) {
			continue
		}

		active = append(
			active,
			configuration,
		)
	}

	if len(ids) > 0 {
		query := s.sq.
			Delete(postgres.TblCredentialsSupported).
			Where(
				squirrel.Eq{
					colTenantId: tenantID,
				},
			).
			Where(
				squirrel.Eq{
					colCredentialConfigurationID: ids,
				},
			)

		sql, params, err := query.ToSql()
		if err != nil {
			return database.NewError(
				"failed to build query",
				err,
			)
		}

		if _, err := s.db.Exec(
			ctx,
			sql,
			params...,
		); err != nil {
			return database.NewError(
				"failed to update credentials supported",
				err,
			)
		}
	}

	if len(active) == 0 {
		return nil
	}

	return s.InsertConfigurationsSupported(
		ctx,
		tenantID,
		active,
	)
}

func (s Store) listIssuers(
	ctx context.Context,
	orderBy string,
	where ...any,
) ([]issuers.Issuer, error) {
	columns := postgres.PrependAll(
		postgres.TblIssuers,

		colTenantId,
		colCredentialIssuer,
		colAuthorizationServers,
		colCredentialEndpoint,
		colNonceEndpoint,
		colDeferredCredentialEndpoint,
		colNotificationEndpoint,
		colCredentialResponseEncryption,
		colDisplay,
		colSignedMetaData,
		colFirstSeen,
		colLastSeen,
	)

	columns = append(
		columns,
		postgres.PrependAll(
			postgres.TblCredentialsSupported,

			colCredentialConfigurationID,
			colFormat,
			colScope,
			colCryptographicBindingMethodsSupported,
			colSigningAlgValuesSupported,
			colCredentialDefinition,
			colProofTypesSupported,
			colCredentialMetadata,
			colVct,
			colSchema,
			colSubject,
			colFirstSeen,
			colLastSeen,
		)...,
	)

	query := s.sq.
		Select(columns...).
		From(postgres.TblIssuers).
		LeftJoin(
			fmt.Sprintf(
				"%s ON %s.%s=%s.%s",
				postgres.TblCredentialsSupported,
				postgres.TblIssuers,
				colTenantId,
				postgres.TblCredentialsSupported,
				colTenantId,
			),
		).
		OrderBy(
			postgres.Prepend(
				postgres.TblIssuers,
				colCredentialIssuer,
			),
		).
		OrderBy(
			postgres.Prepend(
				postgres.TblIssuers,
				orderBy,
			),
		)

	for _, wh := range where {
		query = query.Where(wh)
	}

	sql, params, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := s.db.Query(
		ctx,
		sql,
		params...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []issuers.Issuer
	var previous *issuers.Issuer

	for rows.Next() {
		var issuer issuers.Issuer
		var csr issuers.CredentialSupportedRow

		err := rows.Scan(
			&issuer.TenantID,
			&issuer.CredentialIssuer,
			&issuer.AuthorizationServers,
			&issuer.CredentialEndpoint,
			&issuer.NonceEndpoint,
			&issuer.DeferredCredentialEndpoint,
			&issuer.NotificationEndpoint,
			&issuer.CredentialResponseEncryption,
			&issuer.Display,
			&issuer.SignedMetadata,
			&issuer.FirstSeen,
			&issuer.LastSeen,

			&csr.CredentialConfigurationID,
			&csr.Format,
			&csr.Scope,
			&csr.CryptographicBindingMethodsSupported,
			&csr.CryptographicSigningAlgValuesSupported,
			&csr.CredentialDefinition,
			&csr.ProofTypesSupported,
			&csr.CredentialMetadata,
			&csr.Vct,
			&csr.Schema,
			&csr.Subject,
			&csr.FirstSeen,
			&csr.LastSeen,
		)
		if err != nil {
			s.log.Error(
				err,
				"failed to scan issuer",
			)

			return nil, err
		}

		if csr.CredentialConfigurationID != nil {
			configuration := issuers.CredentialsSupported{
				TenantID: issuer.TenantID,

				CredentialConfigurationID: *csr.CredentialConfigurationID,

				CryptographicBindingMethodsSupported:   csr.CryptographicBindingMethodsSupported,
				CryptographicSigningAlgValuesSupported: csr.CryptographicSigningAlgValuesSupported,

				ProofTypesSupported: csr.ProofTypesSupported,
				CredentialMetadata:  csr.CredentialMetadata,
				Vct:                 csr.Vct,
				Schema:              csr.Schema,
				FirstSeen:           csr.FirstSeen,
				LastSeen:            csr.LastSeen,
			}

			if csr.Format != nil {
				configuration.Format = *csr.Format
			}

			if csr.Scope != nil {
				configuration.Scope = *csr.Scope
			}

			if csr.CredentialDefinition != nil {
				configuration.CredentialDefinition =
					*csr.CredentialDefinition
			}

			if csr.Subject != nil {
				configuration.Subject = *csr.Subject
			}

			issuer.CredentialsSupported =
				[]issuers.CredentialsSupported{
					configuration,
				}
		}

		if previous == nil {
			previous = &issuer
			continue
		}

		if previous.TenantID != issuer.TenantID {
			out = append(
				out,
				*previous,
			)

			previous = &issuer
			continue
		}

		/*
			LEFT JOIN can yield an issuer without a matching
			credential configuration. Do not access index 0
			unless a configuration actually exists.
		*/
		if len(issuer.CredentialsSupported) > 0 {
			previous.CredentialsSupported = append(
				previous.CredentialsSupported,
				issuer.CredentialsSupported...,
			)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if previous != nil {
		out = append(
			out,
			*previous,
		)
	}

	return out, nil
}

func (s Store) listCredentialConfigurations(
	ctx context.Context,
	orderBy string,
	where ...any,
) ([]issuers.CredentialsSupported, error) {
	columns := postgres.PrependAll(
		postgres.TblCredentialsSupported,

		colTenantId,
		colCredentialConfigurationID,
		colFormat,
		colScope,
		colCryptographicBindingMethodsSupported,
		colSigningAlgValuesSupported,
		colCredentialDefinition,
		colProofTypesSupported,
		colCredentialMetadata,
		colVct,
		colSchema,
		colSubject,
		colFirstSeen,
		colLastSeen,
	)

	query := s.sq.
		Select(columns...).
		From(postgres.TblCredentialsSupported).
		OrderBy(
			postgres.Prepend(
				postgres.TblCredentialsSupported,
				colTenantId,
			),
		).
		OrderBy(
			postgres.Prepend(
				postgres.TblCredentialsSupported,
				orderBy,
			),
		)

	for _, wh := range where {
		query = query.Where(wh)
	}

	sql, params, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := s.db.Query(
		ctx,
		sql,
		params...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(
		[]issuers.CredentialsSupported,
		0,
	)

	for rows.Next() {
		var csr issuers.CredentialSupportedRow

		err := rows.Scan(
			&csr.TenantID,
			&csr.CredentialConfigurationID,
			&csr.Format,
			&csr.Scope,
			&csr.CryptographicBindingMethodsSupported,
			&csr.CryptographicSigningAlgValuesSupported,
			&csr.CredentialDefinition,
			&csr.ProofTypesSupported,
			&csr.CredentialMetadata,
			&csr.Vct,
			&csr.Schema,
			&csr.Subject,
			&csr.FirstSeen,
			&csr.LastSeen,
		)
		if err != nil {
			s.log.Error(
				err,
				"failed to scan credential configuration",
			)

			return nil, err
		}

		if csr.CredentialConfigurationID == nil {
			continue
		}

		credentialSupported := issuers.CredentialsSupported{
			TenantID: csr.TenantID,

			CredentialConfigurationID: *csr.CredentialConfigurationID,

			CryptographicBindingMethodsSupported: csr.CryptographicBindingMethodsSupported,

			CryptographicSigningAlgValuesSupported: csr.CryptographicSigningAlgValuesSupported,

			ProofTypesSupported: csr.ProofTypesSupported,

			CredentialMetadata: csr.CredentialMetadata,

			Vct: csr.Vct,

			Schema: csr.Schema,

			FirstSeen: csr.FirstSeen,
			LastSeen:  csr.LastSeen,
		}

		if csr.Format != nil {
			credentialSupported.Format = *csr.Format
		}

		if csr.Scope != nil {
			credentialSupported.Scope = *csr.Scope
		}

		if csr.CredentialDefinition != nil {
			credentialSupported.CredentialDefinition =
				*csr.CredentialDefinition
		}

		if csr.Subject != nil {
			credentialSupported.Subject = *csr.Subject
		}

		out = append(
			out,
			credentialSupported,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return out, nil
}
