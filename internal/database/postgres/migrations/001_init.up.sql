DROP TABLE IF EXISTS credentials_supported;
DROP TABLE IF EXISTS issuers;

CREATE TABLE issuers (
    tenant_id                       text NOT NULL,
    credential_issuer               text NOT NULL,
    authorization_servers           text[],
    credential_endpoint             text NOT NULL,
    nonce_endpoint                  text DEFAULT NULL,
    deferred_credential_endpoint    text DEFAULT NULL,
    notification_endpoint           text DEFAULT NULL,
    credential_response_encryption  jsonb DEFAULT NULL,
    batch_credential_issuance       jsonb DEFAULT NULL,
    display                         jsonb DEFAULT NULL,
    signed_metadata                 text DEFAULT NULL,
    first_seen                      timestamp with time zone,
    last_seen                       timestamp with time zone,

    PRIMARY KEY (tenant_id)
);

CREATE TABLE credentials_supported (
    tenant_id                                  text NOT NULL,
    credential_configuration_id                text NOT NULL,
    format                                     text NOT NULL,
    scope                                      text DEFAULT NULL,
    cryptographic_binding_methods_supported    text[],
    credential_signing_alg_values_supported    text[],
    proof_types_supported                      jsonb DEFAULT NULL,

    credential_definition                      jsonb DEFAULT NULL,
    credential_metadata                        jsonb DEFAULT NULL,

    vct                                        text DEFAULT NULL,

    schema                                     jsonb DEFAULT NULL,
    subject                                    text DEFAULT NULL,

    first_seen                                 timestamp with time zone,
    last_seen                                  timestamp with time zone,

    PRIMARY KEY (tenant_id, credential_configuration_id),

    CONSTRAINT fk_credentials_supported_issuer
        FOREIGN KEY (tenant_id)
        REFERENCES issuers (tenant_id)
        ON DELETE CASCADE
);

CREATE INDEX idx_credentials_supported_tenant_id
    ON credentials_supported (tenant_id);

CREATE INDEX idx_credentials_supported_format
    ON credentials_supported (format);