-- 添加分组-模型绑定表

CREATE TABLE IF NOT EXISTS group_models (
    id TEXT PRIMARY KEY,
    group_id TEXT NOT NULL,
    model_id TEXT NOT NULL,
    enabled INTEGER DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE,
    FOREIGN KEY (model_id) REFERENCES models(id) ON DELETE CASCADE,
    UNIQUE(group_id, model_id)
);

CREATE INDEX IF NOT EXISTS idx_group_models_group ON group_models(group_id);
CREATE INDEX IF NOT EXISTS idx_group_models_model ON group_models(model_id);

CREATE TRIGGER IF NOT EXISTS trg_group_models_updated
AFTER UPDATE ON group_models
BEGIN
    UPDATE group_models SET updated_at = datetime('now') WHERE id = NEW.id;
END;
