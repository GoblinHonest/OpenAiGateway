-- Seed data for development (MySQL)
INSERT IGNORE INTO groups (id, name, description, load_balance_strategy, enabled)
VALUES ('group-default', 'default', 'Default group', 'round_robin', 1);

INSERT IGNORE INTO providers (id, name, base_url, status, supported_formats, endpoints)
VALUES
  ('openai-001', 'OpenAI', 'https://api.openai.com', 'active',
   '["openai"]', '{"chat": "/v1/chat/completions", "embeddings": "/v1/embeddings"}'),
  ('anthropic-001', 'Anthropic', 'https://api.anthropic.com', 'active',
   '["anthropic"]', '{"chat": "/v1/messages"}');

INSERT IGNORE INTO models (id, name, display_name, model_type, context_window, max_output_tokens, enabled)
VALUES
  ('model-gpt4', 'gpt-4', 'GPT-4', 'chat', 8192, 4096, 1),
  ('model-gpt35', 'gpt-3.5-turbo', 'GPT-3.5 Turbo', 'chat', 16384, 4096, 1),
  ('model-claude3', 'claude-3-opus-20240229', 'Claude 3 Opus', 'chat', 200000, 4096, 1);

INSERT IGNORE INTO api_keys (id, key_hash, key_prefix, name, group_id, status)
VALUES ('key-dev', 'dev-hash-placeholder', 'sk-dev', 'Development Key', 'group-default', 'active');
