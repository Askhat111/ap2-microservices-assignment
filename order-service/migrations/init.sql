DROP TABLE IF EXISTS orders;
CREATE TABLE orders (
    id VARCHAR(255) PRIMARY KEY,
    customer_id VARCHAR(255),
    item_name VARCHAR(255),
    amount BIGINT,
    status VARCHAR(50),
    created_at TIMESTAMP,
    idempotency_key VARCHAR(255) UNIQUE
);