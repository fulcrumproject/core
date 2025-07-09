-- Create metrics schema
CREATE SCHEMA IF NOT EXISTS metrics;

-- Grant permissions to the fulcrum user
GRANT ALL PRIVILEGES ON SCHEMA metrics TO fulcrum;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA metrics TO fulcrum;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA metrics TO fulcrum;

-- Set default privileges for future tables
ALTER DEFAULT PRIVILEGES IN SCHEMA metrics GRANT ALL ON TABLES TO fulcrum;
ALTER DEFAULT PRIVILEGES IN SCHEMA metrics GRANT ALL ON SEQUENCES TO fulcrum; 