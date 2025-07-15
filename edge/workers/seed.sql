-- Demo user seed data
INSERT INTO users (
  id, 
  email, 
  password_hash, 
  name, 
  email_verified, 
  subscription_status, 
  subscription_plan,
  api_usage_count,
  created_at,
  updated_at
) VALUES (
  'usr_demo_001',
  'demo@example.com',
  '$2a$10$K7L3gNvl.IkQ.FRQz.y0AO7fVbKDrO0PvnCKI6BXKzKHKeqkr5hO2', -- password: demo1234
  'Demo User',
  true,
  'active',
  'pro',
  250,
  datetime('now'),
  datetime('now')
);

-- Demo domains
INSERT INTO domains (id, user_id, domain, registrar, status, created_at, updated_at) VALUES
('dom_demo_001', 'usr_demo_001', 'example.com', 'value-domain', 'active', datetime('now'), datetime('now')),
('dom_demo_002', 'usr_demo_001', 'demo-site.dev', 'porkbun', 'active', datetime('now'), datetime('now')),
('dom_demo_003', 'usr_demo_001', 'my-app.cloud', 'route53', 'active', datetime('now'), datetime('now'));

-- Demo DNS records
INSERT INTO dns_records (id, domain_id, type, name, value, ttl, created_at, updated_at) VALUES
('dns_demo_001', 'dom_demo_001', 'A', '@', '192.0.2.1', 300, datetime('now'), datetime('now')),
('dns_demo_002', 'dom_demo_001', 'A', 'www', '192.0.2.1', 300, datetime('now'), datetime('now')),
('dns_demo_003', 'dom_demo_001', 'MX', '@', '10 mail.example.com', 300, datetime('now'), datetime('now')),
('dns_demo_004', 'dom_demo_001', 'TXT', '@', 'v=spf1 include:_spf.example.com ~all', 300, datetime('now'), datetime('now')),
('dns_demo_005', 'dom_demo_002', 'A', '@', '198.51.100.1', 300, datetime('now'), datetime('now')),
('dns_demo_006', 'dom_demo_002', 'CNAME', 'www', 'demo-site.dev', 300, datetime('now'), datetime('now')),
('dns_demo_007', 'dom_demo_003', 'A', '@', '203.0.113.1', 300, datetime('now'), datetime('now')),
('dns_demo_008', 'dom_demo_003', 'A', 'api', '203.0.113.2', 300, datetime('now'), datetime('now'));

-- Demo API key
INSERT INTO api_keys (id, user_id, name, key_hash, key_preview, scopes, is_active, created_at) VALUES
('key_demo_001', 'usr_demo_001', 'Demo API Key', 'demo_key_hash', 'demo_****_****', '["read", "write"]', true, datetime('now'));