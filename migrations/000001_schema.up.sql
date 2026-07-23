-- Birleşik şema: tüm tablolar, kısıtlar ve index'ler (güncel final durum).
-- users tablosunda version sütunu yoktur (optimistic locking kullanılmıyor).

-- RBAC (users.role FK için önce roller tablosu).
CREATE TABLE IF NOT EXISTS roles (
    id          UUID PRIMARY KEY,
    name        TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    is_system   BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS permissions (
    id          UUID PRIMARY KEY,
    name        TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS role_permissions (
    role_id       UUID NOT NULL REFERENCES roles (id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions (id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

CREATE INDEX IF NOT EXISTS role_permissions_perm_idx ON role_permissions (permission_id);

-- users: kullanıcı aggregate'inin kalıcı temsili.
CREATE TABLE IF NOT EXISTS users (
    id                 UUID PRIMARY KEY,
    email              TEXT NOT NULL,
    phone              TEXT NOT NULL DEFAULT '',
    name               TEXT NOT NULL,
    password_hash      TEXT NOT NULL,
    role               TEXT NOT NULL DEFAULT 'user',
    active             BOOLEAN NOT NULL DEFAULT FALSE,
    email_verified     BOOLEAN NOT NULL DEFAULT FALSE,
    mfa_enabled        BOOLEAN NOT NULL DEFAULT FALSE,
    mfa_secret         TEXT NOT NULL DEFAULT '',
    mfa_last_used_step BIGINT,
    preferred_locale   TEXT NOT NULL DEFAULT 'tr',
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at         TIMESTAMPTZ,

    CONSTRAINT users_role_fk FOREIGN KEY (role) REFERENCES roles (name) ON UPDATE CASCADE,
    CONSTRAINT users_preferred_locale_chk CHECK (preferred_locale ~ '^[a-z]{2}(-[a-z]{2})?$'),
    CONSTRAINT users_phone_chk CHECK (phone = '' OR phone ~ '^\+[1-9][0-9]{7,14}$')
);

CREATE UNIQUE INDEX IF NOT EXISTS users_email_uidx
    ON users (email)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS users_keyset_idx
    ON users (created_at DESC, id DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS users_role_idx ON users (role);
CREATE INDEX IF NOT EXISTS users_active_idx ON users (active);

-- MFA kurtarma kodları.
CREATE TABLE IF NOT EXISTS mfa_recovery_codes (
    id         UUID PRIMARY KEY,
    user_id    UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    code_hash  TEXT NOT NULL UNIQUE,
    used_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS mfa_recovery_codes_unused_user_idx
    ON mfa_recovery_codes (user_id)
    WHERE used_at IS NULL;

-- audit_logs: değiştirilemez denetim kaydı.
CREATE TABLE IF NOT EXISTS audit_logs (
    id             UUID PRIMARY KEY,
    event_id       UUID,
    actor_id       UUID,
    actor_type     TEXT NOT NULL DEFAULT 'user',
    actor_email    TEXT NOT NULL DEFAULT '',
    action         TEXT NOT NULL,
    resource       TEXT NOT NULL,
    resource_id    TEXT,
    ip             INET,
    user_agent     TEXT,
    metadata       JSONB NOT NULL DEFAULT '{}'::jsonb,
    source         TEXT NOT NULL DEFAULT '',
    correlation_id TEXT NOT NULL DEFAULT '',
    occurred_at    TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS audit_logs_event_id_uidx
    ON audit_logs (event_id)
    WHERE event_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS audit_logs_actor_idx ON audit_logs (actor_id, created_at DESC);
CREATE INDEX IF NOT EXISTS audit_logs_created_idx ON audit_logs (created_at DESC);
CREATE INDEX IF NOT EXISTS audit_logs_resource_idx ON audit_logs (resource, created_at DESC);
CREATE INDEX IF NOT EXISTS audit_logs_action_idx ON audit_logs (action, created_at DESC);

-- notifications: uygulama içi bildirimler.
CREATE TABLE IF NOT EXISTS notifications (
    id           UUID PRIMARY KEY,
    recipient_id UUID NOT NULL,
    title        TEXT NOT NULL,
    content      TEXT NOT NULL,
    read         BOOLEAN NOT NULL DEFAULT FALSE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    read_at      TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS notifications_recipient_idx
    ON notifications (recipient_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS notifications_unread_idx
    ON notifications (recipient_id)
    WHERE read = FALSE;

-- auth_tokens: tek kullanımlık, süreli token'lar (e-posta doğrulama, şifre sıfırlama).
CREATE TABLE IF NOT EXISTS auth_tokens (
    id         UUID PRIMARY KEY,
    user_id    UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    purpose    TEXT NOT NULL,
    token_hash TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT auth_tokens_purpose_chk CHECK (purpose IN ('email_verify', 'password_reset'))
);

CREATE UNIQUE INDEX IF NOT EXISTS auth_tokens_hash_uidx ON auth_tokens (token_hash);
CREATE INDEX IF NOT EXISTS auth_tokens_user_purpose_idx ON auth_tokens (user_id, purpose);

-- app_settings: key-value uygulama ayarları.
CREATE TABLE IF NOT EXISTS app_settings (
    key        TEXT PRIMARY KEY,
    value      TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- payments: 3DS tahsilat kayıtları.
CREATE TABLE IF NOT EXISTS payments (
    id                  UUID PRIMARY KEY,
    reference           TEXT NOT NULL UNIQUE,
    provider            TEXT NOT NULL DEFAULT 'iyzico',
    status              TEXT NOT NULL DEFAULT 'pending',
    stage               TEXT NOT NULL DEFAULT 'initialized',
    amount              TEXT NOT NULL DEFAULT '',
    paid_amount         TEXT NOT NULL DEFAULT '',
    currency            TEXT NOT NULL DEFAULT 'TRY',
    installment         INT NOT NULL DEFAULT 1,
    buyer_name          TEXT NOT NULL DEFAULT '',
    buyer_surname       TEXT NOT NULL DEFAULT '',
    buyer_email         TEXT NOT NULL DEFAULT '',
    buyer_phone         TEXT NOT NULL DEFAULT '',
    card_holder         TEXT NOT NULL DEFAULT '',
    card_bin            TEXT NOT NULL DEFAULT '',
    card_last4          TEXT NOT NULL DEFAULT '',
    card_association    TEXT NOT NULL DEFAULT '',
    provider_payment_id TEXT NOT NULL DEFAULT '',
    result_code         TEXT NOT NULL DEFAULT '',
    result_message      TEXT NOT NULL DEFAULT '',
    auth_code           TEXT NOT NULL DEFAULT '',
    conversation_data   TEXT NOT NULL DEFAULT '',
    init_payload        TEXT NOT NULL DEFAULT '',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at        TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_payments_status ON payments (status);
CREATE INDEX IF NOT EXISTS idx_payments_provider ON payments (provider);
CREATE INDEX IF NOT EXISTS idx_payments_created_at ON payments (created_at DESC);

-- idempotency_keys: ödeme, bildirim ve diğer tekrarlanmaması gereken işlemler.
CREATE TABLE IF NOT EXISTS idempotency_keys (
    id           UUID PRIMARY KEY,
    scope        TEXT NOT NULL,
    key          TEXT NOT NULL,
    actor_id     TEXT NOT NULL DEFAULT '',
    request_hash TEXT NOT NULL DEFAULT '',
    status       TEXT NOT NULL DEFAULT 'processing',
    response     JSONB,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at   TIMESTAMPTZ NOT NULL,

    CONSTRAINT idempotency_keys_status_chk CHECK (status IN ('processing', 'completed', 'failed')),
    CONSTRAINT idempotency_keys_scope_key_actor_uidx UNIQUE (scope, key, actor_id)
);

CREATE INDEX IF NOT EXISTS idempotency_keys_expires_idx ON idempotency_keys (expires_at);

-- Kalıcı iş kuyruğu / transactional outbox.
CREATE TABLE IF NOT EXISTS outbox_jobs (
    id              UUID PRIMARY KEY,
    kind            TEXT NOT NULL,
    aggregate_type  TEXT NOT NULL DEFAULT '',
    aggregate_id    TEXT NOT NULL DEFAULT '',
    idempotency_key TEXT,
    payload         JSONB NOT NULL DEFAULT '{}'::jsonb,
    status          TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'processing', 'completed', 'failed', 'dead')),
    attempts        INT NOT NULL DEFAULT 0,
    max_attempts    INT NOT NULL DEFAULT 8,
    available_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    lease_until     TIMESTAMPTZ,
    last_error      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at    TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS outbox_jobs_idempotency_uidx
    ON outbox_jobs (kind, idempotency_key)
    WHERE idempotency_key IS NOT NULL AND idempotency_key <> '';

CREATE INDEX IF NOT EXISTS outbox_jobs_claim_idx
    ON outbox_jobs (available_at, created_at)
    WHERE status IN ('pending', 'failed');

CREATE INDEX IF NOT EXISTS outbox_jobs_lease_idx
    ON outbox_jobs (lease_until)
    WHERE status = 'processing';

-- İletişim formu mesajları.
CREATE TABLE IF NOT EXISTS contact_messages (
    id         UUID PRIMARY KEY,
    name       TEXT NOT NULL,
    email      TEXT NOT NULL,
    message    TEXT NOT NULL,
    locale     TEXT NOT NULL DEFAULT 'tr',
    ip         INET,
    user_agent TEXT NOT NULL DEFAULT '',
    status     TEXT NOT NULL DEFAULT 'received'
        CHECK (status IN ('received', 'queued', 'notified', 'failed')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    read_at    TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS contact_messages_created_idx
    ON contact_messages (created_at DESC);

CREATE INDEX IF NOT EXISTS contact_messages_unread_idx
    ON contact_messages (created_at DESC)
    WHERE read_at IS NULL;
