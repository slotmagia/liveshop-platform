-- Local development notification templates only. Production must bind
-- templates through the Platform Admin notify-events workflow.
START TRANSACTION;

INSERT INTO platform_notify_template_library
    (code, channel, text_template, subject, body_html, title, body, variables, lifecycle, version, command_key)
VALUES
    (
        'identity.auth.otp.requested.sms',
        'SMS',
        '您的登录验证码是 {{code}}，{{ttlSeconds}} 秒内有效。',
        NULL,
        NULL,
        NULL,
        NULL,
        JSON_ARRAY('code', 'ttlSeconds'),
        'ACTIVE',
        1,
        'seed-local-otp-sms'
    ),
    (
        'identity.auth.otp.requested.email',
        'EMAIL',
        NULL,
        '登录验证码',
        '<p>您的登录验证码是 <strong>{{code}}</strong>，{{ttlSeconds}} 秒内有效。</p>',
        NULL,
        NULL,
        JSON_ARRAY('code', 'ttlSeconds'),
        'ACTIVE',
        1,
        'seed-local-otp-email'
    )
ON DUPLICATE KEY UPDATE
    channel = VALUES(channel),
    text_template = VALUES(text_template),
    subject = VALUES(subject),
    body_html = VALUES(body_html),
    title = VALUES(title),
    body = VALUES(body),
    variables = VALUES(variables),
    lifecycle = 'ACTIVE';

UPDATE platform_notify_event_policy
SET
    template_sms = IF(template_sms = '', 'identity.auth.otp.requested.sms', template_sms),
    template_email = IF(template_email = '', 'identity.auth.otp.requested.email', template_email)
WHERE event_key = 'identity.auth.otp.requested';

COMMIT;
