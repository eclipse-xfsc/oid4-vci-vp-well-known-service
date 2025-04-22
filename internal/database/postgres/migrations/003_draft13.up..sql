ALTER TABLE issuers ADD signed_metadata text DEFAULT NULL;
ALTER TABLE issuers ADD notification_endpoint text DEFAULT NULL;
ALTER TABLE issuers ADD credential_identifiers_supported boolean DEFAULT false;


ALTER TABLE credentials_supported ADD vct   text DEFAULT NULL;
ALTER TABLE credentials_supported ADD claims  jsonb DEFAULT NULL;
ALTER TABLE credentials_supported ADD "order" text[] DEFAULT NULL;