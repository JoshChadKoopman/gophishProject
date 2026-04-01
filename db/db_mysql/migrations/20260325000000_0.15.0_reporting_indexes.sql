-- +goose Up
CREATE INDEX idx_results_campaign_status ON results(campaign_id, status);
CREATE INDEX idx_events_campaign_time ON events(campaign_id, time);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_timestamp ON audit_logs(timestamp);
CREATE INDEX idx_course_progress_status ON course_progress(status);

-- +goose Down
DROP INDEX idx_results_campaign_status ON results;
DROP INDEX idx_events_campaign_time ON events;
DROP INDEX idx_audit_logs_action ON audit_logs;
DROP INDEX idx_audit_logs_timestamp ON audit_logs;
DROP INDEX idx_course_progress_status ON course_progress;
