-- +goose Up
-- Phase 11: Academy Tier Progression & Gamification

-- Academy tiers: Bronze, Silver, Gold, Platinum
CREATE TABLE IF NOT EXISTS academy_tiers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    org_id INTEGER NOT NULL DEFAULT 0,
    slug VARCHAR(50) NOT NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    badge_icon_url VARCHAR(255) NOT NULL DEFAULT '',
    sort_order INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT 1,
    created_date DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_academy_tiers_org_slug ON academy_tiers(org_id, slug);

-- Sessions within a tier (link to existing training_presentations)
CREATE TABLE IF NOT EXISTS academy_sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tier_id INTEGER NOT NULL,
    presentation_id INTEGER NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    estimated_minutes INTEGER NOT NULL DEFAULT 10,
    is_required BOOLEAN NOT NULL DEFAULT 1,
    created_date DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_academy_sessions_tier ON academy_sessions(tier_id);

-- User progress per academy tier
CREATE TABLE IF NOT EXISTS academy_user_progress (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    tier_id INTEGER NOT NULL,
    sessions_completed INTEGER NOT NULL DEFAULT 0,
    tier_unlocked BOOLEAN NOT NULL DEFAULT 0,
    tier_completed BOOLEAN NOT NULL DEFAULT 0,
    completed_date DATETIME,
    created_date DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_academy_progress_user_tier ON academy_user_progress(user_id, tier_id);

-- Compliance certification paths
CREATE TABLE IF NOT EXISTS compliance_certifications (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    org_id INTEGER NOT NULL DEFAULT 0,
    slug VARCHAR(50) NOT NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    required_session_ids TEXT NOT NULL DEFAULT '[]',
    is_active BOOLEAN NOT NULL DEFAULT 1,
    created_date DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_compliance_cert_org_slug ON compliance_certifications(org_id, slug);

-- User compliance certifications
CREATE TABLE IF NOT EXISTS user_compliance_certs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    certification_id INTEGER NOT NULL,
    verification_code VARCHAR(32) NOT NULL,
    issued_date DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_date DATETIME
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_user_compliance_code ON user_compliance_certs(verification_code);
CREATE INDEX IF NOT EXISTS idx_user_compliance_user ON user_compliance_certs(user_id);

-- Badges (achievements)
CREATE TABLE IF NOT EXISTS badges (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    slug VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    icon_url VARCHAR(255) NOT NULL DEFAULT '',
    category VARCHAR(50) NOT NULL DEFAULT 'general',
    criteria_type VARCHAR(50) NOT NULL DEFAULT '',
    criteria_value INTEGER NOT NULL DEFAULT 0,
    created_date DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- User earned badges
CREATE TABLE IF NOT EXISTS user_badges (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    badge_id INTEGER NOT NULL,
    earned_date DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_user_badges_unique ON user_badges(user_id, badge_id);

-- User streaks
CREATE TABLE IF NOT EXISTS user_streaks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    streak_type VARCHAR(50) NOT NULL DEFAULT 'weekly',
    current_streak INTEGER NOT NULL DEFAULT 0,
    longest_streak INTEGER NOT NULL DEFAULT 0,
    last_activity_date DATETIME,
    created_date DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_user_streaks_unique ON user_streaks(user_id, streak_type);

-- Leaderboard cache (recalculated nightly)
CREATE TABLE IF NOT EXISTS leaderboard_cache (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    org_id INTEGER NOT NULL,
    department VARCHAR(100) NOT NULL DEFAULT '',
    user_id INTEGER NOT NULL,
    score INTEGER NOT NULL DEFAULT 0,
    rank INTEGER NOT NULL DEFAULT 0,
    period VARCHAR(20) NOT NULL DEFAULT 'all_time',
    calculated_date DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_leaderboard_org_period ON leaderboard_cache(org_id, period);
CREATE INDEX IF NOT EXISTS idx_leaderboard_user ON leaderboard_cache(user_id);

-- Seed default academy tiers (org_id=0 = system defaults)
INSERT INTO academy_tiers (org_id, slug, name, description, badge_icon_url, sort_order) VALUES
    (0, 'bronze', 'Bronze', 'Foundation level — learn the basics of cybersecurity awareness', '/images/badges/bronze.png', 1),
    (0, 'silver', 'Silver', 'Intermediate level — recognize common attack patterns', '/images/badges/silver.png', 2),
    (0, 'gold', 'Gold', 'Advanced level — defend against sophisticated threats', '/images/badges/gold.png', 3),
    (0, 'platinum', 'Platinum', 'Expert level — become a security champion', '/images/badges/platinum.png', 4);

-- Seed default badges
INSERT INTO badges (slug, name, description, icon_url, category, criteria_type, criteria_value) VALUES
    ('first_course', 'First Steps', 'Completed your first training course', '', 'training', 'courses_completed', 1),
    ('five_courses', 'Dedicated Learner', 'Completed 5 training courses', '', 'training', 'courses_completed', 5),
    ('ten_courses', 'Knowledge Seeker', 'Completed 10 training courses', '', 'training', 'courses_completed', 10),
    ('perfect_quiz', 'Perfect Score', 'Scored 100% on a quiz', '', 'quiz', 'perfect_quiz', 1),
    ('five_quizzes', 'Quiz Master', 'Passed 5 quizzes', '', 'quiz', 'quizzes_passed', 5),
    ('bronze_tier', 'Bronze Graduate', 'Completed the Bronze academy tier', '', 'academy', 'tier_completed', 1),
    ('silver_tier', 'Silver Graduate', 'Completed the Silver academy tier', '', 'academy', 'tier_completed', 2),
    ('gold_tier', 'Gold Graduate', 'Completed the Gold academy tier', '', 'academy', 'tier_completed', 3),
    ('platinum_tier', 'Platinum Graduate', 'Completed the Platinum academy tier', '', 'academy', 'tier_completed', 4),
    ('week_streak_3', 'On a Roll', 'Maintained a 3-week training streak', '', 'streak', 'weekly_streak', 3),
    ('week_streak_8', 'Consistent', 'Maintained an 8-week training streak', '', 'streak', 'weekly_streak', 8),
    ('week_streak_16', 'Unstoppable', 'Maintained a 16-week training streak', '', 'streak', 'weekly_streak', 16),
    ('phish_reporter', 'Eagle Eye', 'Reported a simulated phishing email', '', 'simulation', 'phish_reported', 1),
    ('phish_reporter_10', 'Phish Hunter', 'Reported 10 simulated phishing emails', '', 'simulation', 'phish_reported', 10),
    ('compliance_cert', 'Compliance Champion', 'Earned a compliance certification', '', 'compliance', 'compliance_earned', 1);

-- Seed default compliance certification paths (org_id=0 = templates)
INSERT INTO compliance_certifications (org_id, slug, name, description) VALUES
    (0, 'nis2', 'NIS2 Directive', 'Network and Information Security Directive compliance training'),
    (0, 'gdpr', 'GDPR', 'General Data Protection Regulation awareness training'),
    (0, 'iso27001', 'ISO 27001', 'Information Security Management System awareness'),
    (0, 'dora', 'DORA', 'Digital Operational Resilience Act compliance training'),
    (0, 'hipaa', 'HIPAA', 'Health Insurance Portability and Accountability Act training'),
    (0, 'pci_dss', 'PCI DSS', 'Payment Card Industry Data Security Standard training');

-- +goose Down
DROP TABLE IF EXISTS leaderboard_cache;
DROP TABLE IF EXISTS user_streaks;
DROP TABLE IF EXISTS user_badges;
DROP TABLE IF EXISTS badges;
DROP TABLE IF EXISTS user_compliance_certs;
DROP TABLE IF EXISTS compliance_certifications;
DROP TABLE IF EXISTS academy_user_progress;
DROP TABLE IF EXISTS academy_sessions;
DROP TABLE IF EXISTS academy_tiers;
