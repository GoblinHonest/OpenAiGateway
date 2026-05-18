-- PostgreSQL: Create group_models table
CREATE TABLE IF NOT EXISTS group_models (
    id VARCHAR(64) PRIMARY KEY,
    group_id VARCHAR(64) NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    model_id VARCHAR(64) NOT NULL REFERENCES models(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(group_id, model_id)
);

CREATE INDEX IF NOT EXISTS idx_group_models_group ON group_models(group_id);
CREATE INDEX IF NOT EXISTS idx_group_models_model ON group_models(model_id);
