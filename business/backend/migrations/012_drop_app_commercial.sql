-- Hard cut: Platform catalogues become a single global catalogue (no app_id).
-- SMS merchant grants become merchant_id + shop_id. Settings are global
-- (PLATFORM, merchant_id=0) or merchant-scoped. Historical 001-011 stay unchanged.

DROP PROCEDURE IF EXISTS platform_drop_scope_replace_index;
DROP PROCEDURE IF EXISTS platform_drop_scope_drop_index;
DROP PROCEDURE IF EXISTS platform_drop_scope_drop_pk;
DROP PROCEDURE IF EXISTS platform_drop_scope_drop_check;
DROP PROCEDURE IF EXISTS platform_drop_scope_add_check;
DROP PROCEDURE IF EXISTS platform_drop_scope_drop_column;
DROP PROCEDURE IF EXISTS platform_drop_scope_add_column;
DROP PROCEDURE IF EXISTS platform_drop_scope_assert_zero;

DELIMITER $$
CREATE PROCEDURE platform_drop_scope_replace_index(IN p_table VARCHAR(128), IN p_name VARCHAR(128), IN p_definition TEXT)
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME=p_table AND INDEX_NAME=p_name) THEN
        SET @scope_sql=CONCAT('ALTER TABLE `',p_table,'` DROP INDEX `',p_name,'`');
        PREPARE scope_stmt FROM @scope_sql; EXECUTE scope_stmt; DEALLOCATE PREPARE scope_stmt;
    END IF;
    SET @scope_sql=CONCAT('ALTER TABLE `',p_table,'` ADD ',p_definition);
    PREPARE scope_stmt FROM @scope_sql; EXECUTE scope_stmt; DEALLOCATE PREPARE scope_stmt;
END$$

CREATE PROCEDURE platform_drop_scope_drop_index(IN p_table VARCHAR(128), IN p_name VARCHAR(128))
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME=p_table AND INDEX_NAME=p_name) THEN
        SET @scope_sql=CONCAT('ALTER TABLE `',p_table,'` DROP INDEX `',p_name,'`');
        PREPARE scope_stmt FROM @scope_sql; EXECUTE scope_stmt; DEALLOCATE PREPARE scope_stmt;
    END IF;
END$$

CREATE PROCEDURE platform_drop_scope_drop_pk(IN p_table VARCHAR(128))
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.TABLE_CONSTRAINTS
        WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME=p_table AND CONSTRAINT_TYPE='PRIMARY KEY'
    ) THEN
        SET @scope_sql=CONCAT('ALTER TABLE `',p_table,'` DROP PRIMARY KEY');
        PREPARE scope_stmt FROM @scope_sql; EXECUTE scope_stmt; DEALLOCATE PREPARE scope_stmt;
    END IF;
END$$

CREATE PROCEDURE platform_drop_scope_drop_check(IN p_table VARCHAR(128), IN p_name VARCHAR(128))
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.TABLE_CONSTRAINTS
        WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME=p_table AND CONSTRAINT_NAME=p_name AND CONSTRAINT_TYPE='CHECK'
    ) THEN
        SET @scope_sql=CONCAT('ALTER TABLE `',p_table,'` DROP CHECK `',p_name,'`');
        PREPARE scope_stmt FROM @scope_sql; EXECUTE scope_stmt; DEALLOCATE PREPARE scope_stmt;
    END IF;
END$$

CREATE PROCEDURE platform_drop_scope_add_check(IN p_table VARCHAR(128), IN p_name VARCHAR(128), IN p_expression TEXT)
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.TABLE_CONSTRAINTS
        WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME=p_table AND CONSTRAINT_NAME=p_name AND CONSTRAINT_TYPE='CHECK'
    ) THEN
        SET @scope_sql=CONCAT('ALTER TABLE `',p_table,'` ADD CONSTRAINT `',p_name,'` CHECK (',p_expression,')');
        PREPARE scope_stmt FROM @scope_sql; EXECUTE scope_stmt; DEALLOCATE PREPARE scope_stmt;
    END IF;
END$$

CREATE PROCEDURE platform_drop_scope_drop_column(IN p_table VARCHAR(128), IN p_column VARCHAR(128))
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME=p_table AND COLUMN_NAME=p_column) THEN
        SET @scope_sql=CONCAT('ALTER TABLE `',p_table,'` DROP COLUMN `',p_column,'`');
        PREPARE scope_stmt FROM @scope_sql; EXECUTE scope_stmt; DEALLOCATE PREPARE scope_stmt;
    END IF;
END$$

CREATE PROCEDURE platform_drop_scope_add_column(IN p_table VARCHAR(128), IN p_column VARCHAR(128), IN p_definition TEXT)
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME=p_table AND COLUMN_NAME=p_column) THEN
        SET @scope_sql=CONCAT('ALTER TABLE `',p_table,'` ADD COLUMN `',p_column,'` ',p_definition);
        PREPARE scope_stmt FROM @scope_sql; EXECUTE scope_stmt; DEALLOCATE PREPARE scope_stmt;
    END IF;
END$$

CREATE PROCEDURE platform_drop_scope_assert_zero(IN p_table VARCHAR(128))
BEGIN
    SET @leftover=0;
    SET @scope_sql=CONCAT('SELECT COUNT(*) INTO @leftover FROM `',p_table,'` WHERE `app_id`<>0');
    PREPARE scope_stmt FROM @scope_sql; EXECUTE scope_stmt; DEALLOCATE PREPARE scope_stmt;
    IF @leftover > 0 THEN
        SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='Platform catalogue cut blocked: non-global app_id partition exists';
    END IF;
END$$
DELIMITER ;

-- Fail closed if any catalogue/config still uses a non-zero app_id partition.
CALL platform_drop_scope_assert_zero('sms_catalogue');
CALL platform_drop_scope_assert_zero('sms_region');
CALL platform_drop_scope_assert_zero('sms_region_version');
CALL platform_drop_scope_assert_zero('sms_channel');
CALL platform_drop_scope_assert_zero('sms_channel_version');
CALL platform_drop_scope_assert_zero('sms_command');
CALL platform_drop_scope_assert_zero('email_catalogue');
CALL platform_drop_scope_assert_zero('email_config');
CALL platform_drop_scope_assert_zero('email_config_version');
CALL platform_drop_scope_assert_zero('email_command');
CALL platform_drop_scope_assert_zero('storage_catalogue');
CALL platform_drop_scope_assert_zero('storage_channel');
CALL platform_drop_scope_assert_zero('storage_channel_version');
CALL platform_drop_scope_assert_zero('storage_command');
CALL platform_drop_scope_assert_zero('live_provider_catalogue');
CALL platform_drop_scope_assert_zero('live_provider');
CALL platform_drop_scope_assert_zero('live_provider_version');
CALL platform_drop_scope_assert_zero('live_provider_command');

-- SMS grants: replace grant_app_id + commercial_id with merchant_id + shop_id.
CALL platform_drop_scope_add_column('sms_merchant_grant','merchant_id','BIGINT NULL AFTER app_id');
CALL platform_drop_scope_add_column('sms_merchant_grant','shop_id','BIGINT NULL AFTER merchant_id');
CALL platform_drop_scope_add_column('sms_merchant_grant_version','merchant_id','BIGINT NULL AFTER app_id');
CALL platform_drop_scope_add_column('sms_merchant_grant_version','shop_id','BIGINT NULL AFTER merchant_id');

DROP PROCEDURE IF EXISTS platform_drop_scope_backfill_grants;
DELIMITER $$
CREATE PROCEDURE platform_drop_scope_backfill_grants()
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.COLUMNS
        WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='identity_shop' AND COLUMN_NAME='app_id'
    ) AND EXISTS (
        SELECT 1 FROM information_schema.COLUMNS
        WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='identity_shop' AND COLUMN_NAME='commercial_id'
    ) THEN
        UPDATE sms_merchant_grant g
        JOIN (
            SELECT app_id, commercial_id, MIN(merchant_id) AS merchant_id, MIN(shop_id) AS shop_id, COUNT(*) AS shop_count
            FROM identity_shop
            GROUP BY app_id, commercial_id
            HAVING shop_count = 1
        ) s ON s.app_id=g.grant_app_id AND s.commercial_id=g.commercial_id
        SET g.merchant_id=s.merchant_id, g.shop_id=s.shop_id
        WHERE g.merchant_id IS NULL OR g.shop_id IS NULL;
    END IF;

    UPDATE sms_merchant_grant_version v
    JOIN sms_merchant_grant g
      ON g.app_id=v.app_id AND g.grant_app_id=v.grant_app_id AND g.commercial_id=v.commercial_id
    SET v.merchant_id=g.merchant_id, v.shop_id=g.shop_id
    WHERE v.merchant_id IS NULL OR v.shop_id IS NULL;
END$$
DELIMITER ;
CALL platform_drop_scope_backfill_grants();
DROP PROCEDURE platform_drop_scope_backfill_grants;

CREATE TEMPORARY TABLE platform_sms_grant_rekey_guard (
    leftover INT NOT NULL,
    CONSTRAINT ck_platform_sms_grant_rekey_guard CHECK (leftover = 0)
);
INSERT INTO platform_sms_grant_rekey_guard
SELECT (
    SELECT COUNT(*) FROM sms_merchant_grant WHERE merchant_id IS NULL OR shop_id IS NULL OR merchant_id <= 0 OR shop_id <= 0
) + (
    SELECT COUNT(*) FROM sms_merchant_grant_version WHERE merchant_id IS NULL OR shop_id IS NULL OR merchant_id <= 0 OR shop_id <= 0
);
DROP TEMPORARY TABLE platform_sms_grant_rekey_guard;

CREATE TEMPORARY TABLE platform_sms_grant_unique_guard (
    leftover INT NOT NULL,
    CONSTRAINT ck_platform_sms_grant_unique_guard CHECK (leftover = 0)
);
INSERT INTO platform_sms_grant_unique_guard
SELECT COUNT(*) FROM (
    SELECT merchant_id, shop_id FROM sms_merchant_grant GROUP BY merchant_id, shop_id HAVING COUNT(*) > 1
) d;
DROP TEMPORARY TABLE platform_sms_grant_unique_guard;

ALTER TABLE sms_merchant_grant MODIFY COLUMN merchant_id BIGINT NOT NULL;
ALTER TABLE sms_merchant_grant MODIFY COLUMN shop_id BIGINT NOT NULL;
ALTER TABLE sms_merchant_grant_version MODIFY COLUMN merchant_id BIGINT NOT NULL;
ALTER TABLE sms_merchant_grant_version MODIFY COLUMN shop_id BIGINT NOT NULL;

CALL platform_drop_scope_drop_check('sms_merchant_grant','ck_sms_grant_app');
CALL platform_drop_scope_drop_check('sms_merchant_grant','ck_sms_grant_tenant');
CALL platform_drop_scope_drop_index('sms_merchant_grant','uk_sms_grant_tenant');
CALL platform_drop_scope_drop_pk('sms_merchant_grant_version');
CALL platform_drop_scope_drop_column('sms_merchant_grant','app_id');
CALL platform_drop_scope_drop_column('sms_merchant_grant','grant_app_id');
CALL platform_drop_scope_drop_column('sms_merchant_grant','commercial_id');
CALL platform_drop_scope_drop_column('sms_merchant_grant_version','app_id');
CALL platform_drop_scope_drop_column('sms_merchant_grant_version','grant_app_id');
CALL platform_drop_scope_drop_column('sms_merchant_grant_version','commercial_id');
ALTER TABLE sms_merchant_grant ADD UNIQUE KEY uk_sms_grant_shop (merchant_id, shop_id);
ALTER TABLE sms_merchant_grant_version ADD PRIMARY KEY (merchant_id, shop_id, version);
CALL platform_drop_scope_add_check('sms_merchant_grant','ck_sms_grant_shop','merchant_id > 0 AND shop_id > 0');

-- Convert app_id-keyed catalogues into a singleton (id=1) global catalogue.
CALL platform_drop_scope_add_column('sms_catalogue','id','SMALLINT NOT NULL DEFAULT 1 FIRST');
CALL platform_drop_scope_add_column('email_catalogue','id','SMALLINT NOT NULL DEFAULT 1 FIRST');
CALL platform_drop_scope_add_column('storage_catalogue','id','SMALLINT NOT NULL DEFAULT 1 FIRST');
CALL platform_drop_scope_add_column('live_provider_catalogue','id','SMALLINT NOT NULL DEFAULT 1 FIRST');
CALL platform_drop_scope_add_column('email_config','singleton','TINYINT NOT NULL DEFAULT 1 AFTER id');

CALL platform_drop_scope_drop_check('sms_catalogue','ck_sms_catalogue_app');
CALL platform_drop_scope_drop_check('email_catalogue','ck_email_catalogue_app');
CALL platform_drop_scope_drop_check('storage_catalogue','ck_storage_catalogue_app');
CALL platform_drop_scope_drop_check('live_provider_catalogue','ck_live_provider_catalogue_app');
CALL platform_drop_scope_drop_pk('sms_catalogue');
CALL platform_drop_scope_drop_pk('email_catalogue');
CALL platform_drop_scope_drop_pk('storage_catalogue');
CALL platform_drop_scope_drop_pk('live_provider_catalogue');
CALL platform_drop_scope_drop_column('sms_catalogue','app_id');
CALL platform_drop_scope_drop_column('email_catalogue','app_id');
CALL platform_drop_scope_drop_column('storage_catalogue','app_id');
CALL platform_drop_scope_drop_column('live_provider_catalogue','app_id');
ALTER TABLE sms_catalogue ADD PRIMARY KEY (id);
ALTER TABLE email_catalogue ADD PRIMARY KEY (id);
ALTER TABLE storage_catalogue ADD PRIMARY KEY (id);
ALTER TABLE live_provider_catalogue ADD PRIMARY KEY (id);
CALL platform_drop_scope_add_check('sms_catalogue','ck_sms_catalogue_singleton','id = 1');
CALL platform_drop_scope_add_check('email_catalogue','ck_email_catalogue_singleton','id = 1');
CALL platform_drop_scope_add_check('storage_catalogue','ck_storage_catalogue_singleton','id = 1');
CALL platform_drop_scope_add_check('live_provider_catalogue','ck_live_provider_catalogue_singleton','id = 1');

INSERT INTO sms_catalogue(id, revision) VALUES (1, 0)
ON DUPLICATE KEY UPDATE id=VALUES(id);
INSERT INTO email_catalogue(id, revision) VALUES (1, 0)
ON DUPLICATE KEY UPDATE id=VALUES(id);
INSERT INTO storage_catalogue(id, revision) VALUES (1, 0)
ON DUPLICATE KEY UPDATE id=VALUES(id);
INSERT INTO live_provider_catalogue(id, revision) VALUES (1, 0)
ON DUPLICATE KEY UPDATE id=VALUES(id);

-- SMS / email / storage / live-provider facts: unique on code, not app_id+code.
CALL platform_drop_scope_drop_check('sms_region','ck_sms_region_app');
CALL platform_drop_scope_replace_index('sms_region','uk_sms_region_app_code','UNIQUE KEY `uk_sms_region_code` (`code`)');
CALL platform_drop_scope_replace_index('sms_region','uk_sms_region_app_dial','UNIQUE KEY `uk_sms_region_dial` (`dial_code`)');
CALL platform_drop_scope_replace_index('sms_region','idx_sms_region_app_state','KEY `idx_sms_region_state` (`lifecycle`,`sort_order`,`id`)');
CALL platform_drop_scope_drop_column('sms_region','app_id');

CALL platform_drop_scope_drop_pk('sms_region_version');
CALL platform_drop_scope_drop_column('sms_region_version','app_id');
ALTER TABLE sms_region_version ADD PRIMARY KEY (region_code, version);

CALL platform_drop_scope_drop_check('sms_channel','ck_sms_channel_app');
CALL platform_drop_scope_replace_index('sms_channel','uk_sms_channel_app_code','UNIQUE KEY `uk_sms_channel_code` (`code`)');
CALL platform_drop_scope_replace_index('sms_channel','idx_sms_channel_app_state','KEY `idx_sms_channel_state` (`lifecycle`,`enabled`,`id`)');
CALL platform_drop_scope_drop_column('sms_channel','app_id');

CALL platform_drop_scope_drop_pk('sms_channel_version');
CALL platform_drop_scope_drop_column('sms_channel_version','app_id');
ALTER TABLE sms_channel_version ADD PRIMARY KEY (channel_code, version);

CALL platform_drop_scope_drop_pk('sms_command');
CALL platform_drop_scope_drop_column('sms_command','app_id');
ALTER TABLE sms_command ADD PRIMARY KEY (command_key);

CALL platform_drop_scope_drop_check('email_config','ck_email_config_app');
CALL platform_drop_scope_drop_index('email_config','uk_email_config_app');
CALL platform_drop_scope_drop_column('email_config','app_id');
ALTER TABLE email_config ADD UNIQUE KEY uk_email_config_singleton (singleton);
CALL platform_drop_scope_add_check('email_config','ck_email_config_singleton','singleton = 1');

CALL platform_drop_scope_drop_pk('email_config_version');
CALL platform_drop_scope_drop_column('email_config_version','app_id');
ALTER TABLE email_config_version ADD PRIMARY KEY (version);

CALL platform_drop_scope_drop_pk('email_command');
CALL platform_drop_scope_drop_column('email_command','app_id');
ALTER TABLE email_command ADD PRIMARY KEY (command_key);

CALL platform_drop_scope_drop_check('storage_channel','ck_storage_channel_app');
CALL platform_drop_scope_replace_index('storage_channel','uk_storage_channel_app_code','UNIQUE KEY `uk_storage_channel_code` (`code`)');
CALL platform_drop_scope_replace_index('storage_channel','idx_storage_channel_app_state','KEY `idx_storage_channel_state` (`lifecycle`,`is_default`,`enabled`,`id`)');
CALL platform_drop_scope_drop_column('storage_channel','app_id');

CALL platform_drop_scope_drop_pk('storage_channel_version');
CALL platform_drop_scope_drop_column('storage_channel_version','app_id');
ALTER TABLE storage_channel_version ADD PRIMARY KEY (channel_code, version);

CALL platform_drop_scope_drop_pk('storage_command');
CALL platform_drop_scope_drop_column('storage_command','app_id');
ALTER TABLE storage_command ADD PRIMARY KEY (command_key);

CALL platform_drop_scope_drop_check('live_provider','ck_live_provider_app');
CALL platform_drop_scope_replace_index('live_provider','uk_live_provider_app_code','UNIQUE KEY `uk_live_provider_code` (`code`)');
CALL platform_drop_scope_replace_index('live_provider','idx_live_provider_app_state','KEY `idx_live_provider_state` (`lifecycle`,`is_default`,`id`)');
CALL platform_drop_scope_drop_column('live_provider','app_id');

CALL platform_drop_scope_drop_pk('live_provider_version');
CALL platform_drop_scope_drop_column('live_provider_version','app_id');
ALTER TABLE live_provider_version ADD PRIMARY KEY (provider_code, version);

CALL platform_drop_scope_drop_pk('live_provider_command');
CALL platform_drop_scope_drop_column('live_provider_command','app_id');
ALTER TABLE live_provider_command ADD PRIMARY KEY (command_key);

-- Settings: PLATFORM is global (merchant_id=0); MERCHANT is merchant_id-scoped.
UPDATE platform_setting SET merchant_id=0 WHERE realm='PLATFORM';
CREATE TEMPORARY TABLE platform_setting_unique_guard (
    leftover INT NOT NULL,
    CONSTRAINT ck_platform_setting_unique_guard CHECK (leftover = 0)
);
INSERT INTO platform_setting_unique_guard
SELECT COUNT(*) FROM (
    SELECT realm, merchant_id, namespace FROM platform_setting GROUP BY realm, merchant_id, namespace HAVING COUNT(*) > 1
) d;
DROP TEMPORARY TABLE platform_setting_unique_guard;

CALL platform_drop_scope_drop_check('platform_setting','ck_setting_app');
CALL platform_drop_scope_drop_check('platform_setting','ck_setting_merchant');
CALL platform_drop_scope_drop_pk('platform_setting');
CALL platform_drop_scope_drop_column('platform_setting','app_id');
ALTER TABLE platform_setting ADD PRIMARY KEY (realm, merchant_id, namespace);
CALL platform_drop_scope_add_check('platform_setting','ck_setting_scope','(realm = ''PLATFORM'' AND merchant_id = 0) OR (realm = ''MERCHANT'' AND merchant_id > 0)');

CALL platform_drop_scope_drop_index('platform_audit_event','idx_platform_audit_scope_time');
CALL platform_drop_scope_drop_column('platform_audit_event','app_id');
ALTER TABLE platform_audit_event ADD KEY idx_platform_audit_scope_time (realm, merchant_id, occurred_at DESC);

DROP PROCEDURE platform_drop_scope_replace_index;
DROP PROCEDURE platform_drop_scope_drop_index;
DROP PROCEDURE platform_drop_scope_drop_pk;
DROP PROCEDURE platform_drop_scope_drop_check;
DROP PROCEDURE platform_drop_scope_add_check;
DROP PROCEDURE platform_drop_scope_drop_column;
DROP PROCEDURE platform_drop_scope_add_column;
DROP PROCEDURE platform_drop_scope_assert_zero;
