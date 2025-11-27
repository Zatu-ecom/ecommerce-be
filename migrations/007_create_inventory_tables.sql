CREATE TABLE IF NOT EXISTS locations (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(20) DEFAULT 'WAREHOUSE',
    is_active BOOLEAN DEFAULT TRUE,
    priority INTEGER DEFAULT 0,
    address_id INTEGER NOT NULL REFERENCES addresses(id)
);

-- Create inventories table
CREATE TABLE IF NOT EXISTS inventories (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    variant_id INTEGER NOT NULL,
    location_id INTEGER NOT NULL REFERENCES locations(id),
    quantity INTEGER DEFAULT 0 CHECK (quantity >= 0),
    reserved_quantity INTEGER DEFAULT 0 CHECK (reserved_quantity >= 0),
    bin_location VARCHAR(50),
    low_stock_threshold INTEGER DEFAULT 10,
    is_active BOOLEAN DEFAULT TRUE
);

CREATE UNIQUE INDEX idx_inv_var_loc ON inventories(variant_id, location_id);

-- Create inventory_transactions table
CREATE TABLE IF NOT EXISTS inventory_transactions (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    inventory_id INTEGER NOT NULL REFERENCES inventories(id),
    type VARCHAR(20) NOT NULL,
    quantity INTEGER NOT NULL,
    before_quantity INTEGER NOT NULL,
    after_quantity INTEGER NOT NULL,
    reference_id VARCHAR(255),
    reference_type VARCHAR(50),
    note TEXT
);

CREATE INDEX idx_inv_txn_inventory_id ON inventory_transactions(inventory_id);
CREATE INDEX idx_inv_txn_reference ON inventory_transactions(reference_id, reference_type);

-- Create inventory_reservations table
CREATE TABLE IF NOT EXISTS inventory_reservations (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    inventory_id INTEGER NOT NULL REFERENCES inventories(id),
    reference_id VARCHAR(255) NOT NULL,
    quantity INTEGER NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    status VARCHAR(20) DEFAULT 'PENDING'
);

CREATE INDEX idx_inv_res_inventory_id ON inventory_reservations(inventory_id);
CREATE INDEX idx_inv_res_reference_id ON inventory_reservations(reference_id);
CREATE INDEX idx_inv_res_expires_at ON inventory_reservations(expires_at);

-- Create stock_transfers table
CREATE TABLE IF NOT EXISTS stock_transfers (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    reference_number VARCHAR(255) UNIQUE,
    from_location_id INTEGER NOT NULL REFERENCES locations(id),
    to_location_id INTEGER NOT NULL REFERENCES locations(id),
    status VARCHAR(20) DEFAULT 'PENDING',
    requested_by INTEGER,
    shipped_at TIMESTAMPTZ,
    received_at TIMESTAMPTZ
);

-- Create stock_transfer_items table
CREATE TABLE IF NOT EXISTS stock_transfer_items (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    stock_transfer_id INTEGER NOT NULL REFERENCES stock_transfers(id),
    variant_id INTEGER NOT NULL,
    quantity INTEGER NOT NULL
);

CREATE INDEX idx_st_items_transfer_id ON stock_transfer_items(stock_transfer_id);
