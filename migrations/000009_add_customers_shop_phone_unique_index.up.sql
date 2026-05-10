CREATE UNIQUE INDEX idx_customers_shop_id_phone_unique
ON customers (shop_id, phone)
WHERE deleted_at IS NULL;
