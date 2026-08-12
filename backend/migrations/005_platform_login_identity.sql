DO $$
BEGIN
    IF EXISTS (
        SELECT LOWER(username)
        FROM platform_identity_account
        WHERE realm = 'PLATFORM'
        GROUP BY LOWER(username)
        HAVING COUNT(*) > 1
    ) THEN
        RAISE EXCEPTION 'duplicate PLATFORM usernames must be resolved before migration 005';
    END IF;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS uq_platform_identity_global_username
    ON platform_identity_account (LOWER(username))
    WHERE realm = 'PLATFORM';
