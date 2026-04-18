-- Migration: rename group to provider for clarity
-- channels.group -> channels.provider
-- abilities.group -> abilities.provider

-- Rename channels.group to provider
ALTER TABLE channels RENAME COLUMN `group` TO provider;

-- Rename abilities.group to provider
ALTER TABLE abilities RENAME COLUMN `group` TO provider;
