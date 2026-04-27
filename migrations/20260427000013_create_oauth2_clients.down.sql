-- Migration: Drop oauth2 tables
-- Issue: #38

DROP TABLE IF EXISTS oauth2_authorization_codes;
DROP TABLE IF EXISTS oauth2_clients;