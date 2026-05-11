CREATE TABLE ledger_entries (
    id BIGSERIAL PRIMARY KEY,
    shop_id BIGINT NOT NULL REFERENCES shops(id),
    customer_id BIGINT NOT NULL REFERENCES customers(id),
    entry_type TEXT NOT NULL CHECK (entry_type IN ('credit', 'payment', 'adjustment')),
    amount NUMERIC(12, 2) NOT NULL,
    note TEXT,
    transaction_date DATE NOT NULL DEFAULT CURRENT_DATE,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'cancelled')),
    created_by BIGINT NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_ledger_entries_shop_id ON ledger_entries (shop_id);
CREATE INDEX idx_ledger_entries_customer_id ON ledger_entries (customer_id);
CREATE INDEX idx_ledger_entries_shop_customer ON ledger_entries (shop_id, customer_id);
