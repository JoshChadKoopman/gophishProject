-- +goose Up
CREATE INDEX IF NOT EXISTS "idx_results_campaign_status" ON "results"("campaign_id", "status");
CREATE INDEX IF NOT EXISTS "idx_events_campaign_time" ON "events"("campaign_id", "time");
CREATE INDEX IF NOT EXISTS "idx_audit_logs_action" ON "audit_logs"("action");
CREATE INDEX IF NOT EXISTS "idx_audit_logs_timestamp" ON "audit_logs"("timestamp");
CREATE INDEX IF NOT EXISTS "idx_course_progress_status" ON "course_progress"("status");

-- +goose Down
DROP INDEX IF EXISTS "idx_results_campaign_status";
DROP INDEX IF EXISTS "idx_events_campaign_time";
DROP INDEX IF EXISTS "idx_audit_logs_action";
DROP INDEX IF EXISTS "idx_audit_logs_timestamp";
DROP INDEX IF EXISTS "idx_course_progress_status";
