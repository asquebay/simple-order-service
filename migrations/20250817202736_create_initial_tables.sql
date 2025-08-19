-- +goose Up
-- +goose StatementBegin
CREATE TABLE orders (
    order_uid TEXT PRIMARY KEY,
    track_number TEXT NOT NULL,
    entry TEXT,
    locale TEXT,
    internal_signature TEXT,
    customer_id TEXT NOT NULL,
    delivery_service TEXT,
    shardkey TEXT,
    sm_id INT,
    date_created TIMESTAMPTZ NOT NULL,
    oof_shard TEXT
);

CREATE TABLE deliveries (
    order_uid TEXT PRIMARY KEY REFERENCES orders(order_uid) ON DELETE CASCADE,
    name TEXT,
    phone TEXT,
    zip TEXT NOT NULL,
    city TEXT NOT NULL,
    address TEXT NOT NULL,
    region TEXT,
    email TEXT
);

CREATE TABLE payments (
    transaction_uid TEXT PRIMARY KEY REFERENCES orders(order_uid) ON DELETE CASCADE,
    request_id TEXT,
    currency TEXT NOT NULL,
    provider TEXT,
    amount INT NOT NULL,
    payment_dt BIGINT NOT NULL,
    bank TEXT,
    delivery_cost INT NOT NULL,
    goods_total INT NOT NULL,
    custom_fee INT
);

CREATE TABLE items (
    id SERIAL PRIMARY KEY,
    order_uid TEXT NOT NULL REFERENCES orders(order_uid) ON DELETE CASCADE,
    chrt_id BIGINT NOT NULL,
    track_number TEXT NOT NULL,
    price INT NOT NULL,
    rid TEXT,
    name TEXT,
    sale INT,
    size TEXT,
    total_price INT,
    nm_id BIGINT,
    brand TEXT,
    status INT
);

CREATE INDEX idx_items_order_uid ON items (order_uid);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS items;
DROP TABLE IF EXISTS payments;
DROP TABLE IF EXISTS deliveries;
DROP TABLE IF EXISTS orders;
-- +goose StatementEnd
