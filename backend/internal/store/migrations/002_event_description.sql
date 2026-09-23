-- Keep the source catalog text when records are loaded from PostgreSQL.
ALTER TABLE careerquest.events ADD COLUMN description text NOT NULL DEFAULT '';

CREATE TABLE careerquest.legacy_imports (
    source_sha256 text PRIMARY KEY CHECK (length(source_sha256) = 64),
    revision bigint NOT NULL,
    imported_at timestamptz NOT NULL DEFAULT now()
);
