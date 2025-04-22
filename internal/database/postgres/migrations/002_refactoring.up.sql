DROP TABLE IF EXISTS issuers;

CREATE TABLE issuers (
     tenant_id text NOT NULL,
     credential_issuer text NOT NULL,
     authorization_servers text[],
     credential_endpoint text NOT NULL,
     batch_credential_endpoint text DEFAULT NULL,
     deferred_credential_endpoint text DEFAULT NULL,
     credential_response_encryption jsonb DEFAULT NULL,
     display jsonb,
     first_seen timestamp with time zone,
     last_seen timestamp with time zone
);

DROP TABLE IF EXISTS credentials_supported;

CREATE TABLE credentials_supported (
    tenant_id text NOT NULL,
    credential_configuration_id text NOT NULL,
    format text NOT NULL,
    scope text,
    cryptographic_binding_methods_supported text[],
    credential_signing_alg_values_supported text[],
    credential_definition jsonb,
    proof_types_supported jsonb,
    display jsonb,
    schema jsonb,
    subject text,
    first_seen timestamp with time zone,
    last_seen timestamp with time zone
    );