
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied

-- Stores per-user TOTP device state. One device per user (UNIQUE on user_id).
-- totp_secret is stored AES-256-GCM encrypted, base64-encoded.
CREATE TABLE IF NOT EXISTS "mfa_devices" (
    "id"          INTEGER PRIMARY KEY AUTOINCREMENT,
    "user_id"     INTEGER NOT NULL UNIQUE,
    "totp_secret" VARCHAR(512) NOT NULL,
    "enabled"     BOOLEAN NOT NULL DEFAULT 0,
    "enrolled_at" DATETIME,
    FOREIGN KEY ("user_id") REFERENCES "users"("id") ON DELETE CASCADE
);

-- Stores bcrypt-hashed one-time backup codes (8 per user by default).
-- Codes are single-use; once used, used=1 and used_at is set.
CREATE TABLE IF NOT EXISTS "mfa_backup_codes" (
    "id"         INTEGER PRIMARY KEY AUTOINCREMENT,
    "user_id"    INTEGER NOT NULL,
    "code_hash"  VARCHAR(255) NOT NULL,
    "used"       BOOLEAN NOT NULL DEFAULT 0,
    "used_at"    DATETIME,
    "created_at" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY ("user_id") REFERENCES "users"("id") ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS "idx_mfa_backup_codes_user" ON "mfa_backup_codes"("user_id");

-- Records every MFA attempt (success and failure) for lockout enforcement.
-- Lockout: 5 failures within MFALockoutDuration (15 min) blocks further attempts.
CREATE TABLE IF NOT EXISTS "mfa_attempts" (
    "id"           INTEGER PRIMARY KEY AUTOINCREMENT,
    "user_id"      INTEGER NOT NULL,
    "attempted_at" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "success"      BOOLEAN NOT NULL DEFAULT 0,
    "ip_address"   VARCHAR(45)
);
CREATE INDEX IF NOT EXISTS "idx_mfa_attempts_user_time" ON "mfa_attempts"("user_id", "attempted_at");

-- Stores device fingerprint hashes for "Remember this device for 30 days".
-- fingerprint_hash is a SHA-256 of browser/device attributes, bcrypt-stretched.
CREATE TABLE IF NOT EXISTS "device_fingerprints" (
    "id"               INTEGER PRIMARY KEY AUTOINCREMENT,
    "user_id"          INTEGER NOT NULL,
    "fingerprint_hash" VARCHAR(255) NOT NULL,
    "expires_at"       DATETIME NOT NULL,
    "created_at"       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY ("user_id") REFERENCES "users"("id") ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS "idx_device_fingerprints_user" ON "device_fingerprints"("user_id");

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back

DROP TABLE IF EXISTS "device_fingerprints";
DROP TABLE IF EXISTS "mfa_attempts";
DROP TABLE IF EXISTS "mfa_backup_codes";
DROP TABLE IF EXISTS "mfa_devices";
