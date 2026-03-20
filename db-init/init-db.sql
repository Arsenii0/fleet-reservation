-- Enable UUID generation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Resources table (matches the GORM Resource model)
CREATE TABLE IF NOT EXISTS resources (
    resource_id      UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name             TEXT NOT NULL,
    operating_system TEXT NOT NULL,
    CONSTRAINT uni_resources_name UNIQUE (name)
);

-- Seed initial resources
INSERT INTO resources (name, operating_system)
    VALUES ('OpenClaw', 'Ubuntu 24.04')
    ON CONFLICT (name) DO NOTHING;

INSERT INTO resources (name, operating_system)
    VALUES ('VNC', 'Ubuntu 24.04')
    ON CONFLICT (name) DO NOTHING;
