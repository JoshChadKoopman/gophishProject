-- +goose Up
-- Phase 11: Academy Tier Progression & Gamification

CREATE TABLE IF NOT EXISTS academy_tiers (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    org_id BIGINT NOT NULL DEFAULT 0,
    slug VARCHAR(50) NOT NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT NOT NULL,
    badge_icon_url VARCHAR(255) NOT NULL DEFAULT '',
    sort_order INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT 1,
    created_date DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY idx_academy_tiers_org_slug (org_id, slug)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS academy_sessions (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    tier_id BIGINT NOT NULL,
    presentation_id BIGINT NOT NULL,
    sort_order INT NOT NULL DEFAULT 0,
    estimated_minutes INT NOT NULL DEFAULT 10,
    is_required BOOLEAN NOT NULL DEFAULT 1,
    created_date DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_academy_sessions_tier (tier_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS academy_user_progress (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    tier_id BIGINT NOT NULL,
    sessions_completed INT NOT NULL DEFAULT 0,
    tier_unlocked BOOLEAN NOT NULL DEFAULT 0,
    tier_completed BOOLEAN NOT NULL DEFAULT 0,
    completed_date DATETIME,
    created_date DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY idx_academy_progress_user_tier (user_id, tier_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS compliance_certifications (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    org_id BIGINT NOT NULL DEFAULT 0,
    slug VARCHAR(50) NOT NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT NOT NULL,
    required_session_ids TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT 1,
    created_date DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY idx_compliance_cert_org_slug (org_id, slug)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS user_compliance_certs (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    certification_id BIGINT NOT NULL,
    verification_code VARCHAR(32) NOT NULL,
    issued_date DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_date DATETIME,
    UNIQUE KEY idx_user_compliance_code (verification_code),
    INDEX idx_user_compliance_user (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS badges (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    slug VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    description TEXT NOT NULL,
    icon_url VARCHAR(255) NOT NULL DEFAULT '',
    category VARCHAR(50) NOT NULL DEFAULT 'general',
    criteria_type VARCHAR(50) NOT NULL DEFAULT '',
    criteria_value INT NOT NULL DEFAULT 0,
    created_date DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS user_badges (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    badge_id BIGINT NOT NULL,
    earned_date DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY idx_user_badges_unique (user_id, badge_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS user_streaks (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    streak_type VARCHAR(50) NOT NULL DEFAULT 'weekly',
    current_streak INT NOT NULL DEFAULT 0,
    longest_streak INT NOT NULL DEFAULT 0,
    last_activity_date DATETIME,
    created_date DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY idx_user_streaks_unique (user_id, streak_type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS leaderboard_cache (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    org_id BIGINT NOT NULL,
    department VARCHAR(100) NOT NULL DEFAULT '',
    user_id BIGINT NOT NULL,
    score INT NOT NULL DEFAULT 0,
    `rank` INT NOT NULL DEFAULT 0,
    period VARCHAR(20) NOT NULL DEFAULT 'all_time',
    calculated_date DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_leaderboard_org_period (org_id, period),
    INDEX idx_leaderboard_user (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Seed default academy tiers
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

-- Seed default compliance certification paths
INSERT INTO compliance_certifications (org_id, slug, name, description, required_session_ids) VALUES
    (0, 'nis2', 'NIS2 Directive', 'Network and Information Security Directive compliance training', '[]'),
    (0, 'gdpr', 'GDPR', 'General Data Protection Regulation awareness training', '[]'),
    (0, 'iso27001', 'ISO 27001', 'Information Security Management System awareness', '[]'),
    (0, 'dora', 'DORA', 'Digital Operational Resilience Act compliance training', '[]'),
    (0, 'hipaa', 'HIPAA', 'Health Insurance Portability and Accountability Act training', '[]'),
    (0, 'pci_dss', 'PCI DSS', 'Payment Card Industry Data Security Standard training', '[]');

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
