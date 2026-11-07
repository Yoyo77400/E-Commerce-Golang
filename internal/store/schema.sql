CREATE TABLE IF NOT EXISTS users (
    id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role          TEXT NOT NULL DEFAULT 'customer' CHECK (role IN ('customer', 'admin')),
    confirmed_at  TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS user_codes (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    kind       TEXT NOT NULL CHECK (kind IN ('confirm', 'reset')),
    code       TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at    TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS sessions (
    token      TEXT PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS products (
    id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    reference       TEXT NOT NULL UNIQUE,
    name            TEXT NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    category        TEXT NOT NULL,
    price_cents     BIGINT NOT NULL CHECK (price_cents >= 0),
    vat_rate_bp     INT NOT NULL CHECK (vat_rate_bp >= 0),
    price_ttc_cents BIGINT GENERATED ALWAYS AS ((price_cents * (10000 + vat_rate_bp) + 5000) / 10000) STORED,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS carts (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    reference  TEXT NOT NULL UNIQUE,
    user_id    BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    status     TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'paid')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS carts_one_open_per_user ON carts (user_id) WHERE status = 'open';

CREATE TABLE IF NOT EXISTS cart_items (
    cart_id    BIGINT NOT NULL REFERENCES carts (id) ON DELETE CASCADE,
    product_id BIGINT NOT NULL REFERENCES products (id),
    quantity   INT NOT NULL CHECK (quantity > 0),
    PRIMARY KEY (cart_id, product_id)
);

CREATE TABLE IF NOT EXISTS orders (
    id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    reference     TEXT NOT NULL UNIQUE,
    user_id       BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    cart_id       BIGINT REFERENCES carts (id) ON DELETE SET NULL,
    status        TEXT NOT NULL DEFAULT 'pending'
                  CHECK (status IN ('pending', 'paid', 'shipping', 'delivered', 'cancelled')),
    cancel_reason TEXT,
    card_last4    TEXT CHECK (card_last4 ~ '^[0-9]{4}$'),
    paid_at       TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (status <> 'cancelled' OR cancel_reason IS NOT NULL)
);

CREATE INDEX IF NOT EXISTS orders_user_id ON orders (user_id);

CREATE TABLE IF NOT EXISTS order_items (
    order_id         BIGINT NOT NULL REFERENCES orders (id) ON DELETE CASCADE,
    product_id       BIGINT NOT NULL REFERENCES products (id),
    quantity         INT NOT NULL CHECK (quantity > 0),
    unit_price_cents BIGINT NOT NULL CHECK (unit_price_cents >= 0),
    vat_rate_bp      INT NOT NULL CHECK (vat_rate_bp >= 0),
    PRIMARY KEY (order_id, product_id)
);
