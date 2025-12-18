CREATE TABLE filtering_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE,
    expression TEXT NOT NULL,
    enabled BOOLEAN DEFAULT true,
    priority INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_filtering_rules_enabled ON filtering_rules(enabled);
CREATE INDEX idx_filtering_rules_priority ON filtering_rules(priority DESC);
CREATE INDEX idx_filtering_rules_updated_at ON filtering_rules(updated_at DESC);

CREATE TABLE rule_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id UUID NOT NULL,
    rule_type VARCHAR(50) NOT NULL CHECK (rule_type IN ('filtering', 'enrichment', 'dedup')),
    rule_data JSONB NOT NULL,
    version INTEGER NOT NULL,
    changed_by VARCHAR(255),
    change_reason TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(rule_id, version),
    CONSTRAINT fk_rule_versions_filtering_rules 
        FOREIGN KEY (rule_id) REFERENCES filtering_rules(id) ON DELETE CASCADE
);

CREATE INDEX idx_rule_versions_rule_id ON rule_versions(rule_id);
CREATE INDEX idx_rule_versions_rule_type ON rule_versions(rule_type);
CREATE INDEX idx_rule_versions_created_at ON rule_versions(created_at DESC);

CREATE TABLE rule_audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id UUID,
    rule_type VARCHAR(50) NOT NULL CHECK (rule_type IN ('filtering', 'enrichment', 'dedup')),
    action VARCHAR(50) NOT NULL CHECK (action IN ('create', 'update', 'delete', 'toggle')),
    old_value JSONB,
    new_value JSONB,
    changed_by VARCHAR(255) NOT NULL,
    change_reason TEXT,
    ip_address VARCHAR(45),
    timestamp TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_rule_id ON rule_audit_logs(rule_id);
CREATE INDEX idx_audit_logs_rule_type ON rule_audit_logs(rule_type);
CREATE INDEX idx_audit_logs_action ON rule_audit_logs(action);
CREATE INDEX idx_audit_logs_timestamp ON rule_audit_logs(timestamp DESC);
CREATE INDEX idx_audit_logs_changed_by ON rule_audit_logs(changed_by);

CREATE TABLE api_access_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255),
    method VARCHAR(10) NOT NULL,
    path VARCHAR(512) NOT NULL,
    status_code INTEGER,
    response_time_ms INTEGER,
    request_id VARCHAR(255),
    ip_address VARCHAR(45),
    user_agent TEXT,
    timestamp TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_api_logs_user_id ON api_access_logs(user_id);
CREATE INDEX idx_api_logs_timestamp ON api_access_logs(timestamp DESC);
CREATE INDEX idx_api_logs_path ON api_access_logs(path);
CREATE INDEX idx_api_logs_status_code ON api_access_logs(status_code);
