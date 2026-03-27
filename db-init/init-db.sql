-- Enable UUID generation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Resources table (matches the GORM Resource model)
CREATE TABLE IF NOT EXISTS resources (
    resource_id      UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name             TEXT NOT NULL,
    operating_system TEXT NOT NULL,
    plugin           TEXT NOT NULL DEFAULT '',
    CONSTRAINT uni_resources_name UNIQUE (name)
);

-- Seed initial resources
INSERT INTO resources (name, operating_system, plugin)
    VALUES ('OpenClaw', 'Ubuntu 24.04', 'terraform/openclaw-guardian')
    ON CONFLICT (name) DO UPDATE SET plugin = EXCLUDED.plugin;

INSERT INTO resources (name, operating_system, plugin)
    VALUES ('VNC', 'Ubuntu 24.04', '')
    ON CONFLICT (name) DO NOTHING;
