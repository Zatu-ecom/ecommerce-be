CREATE TABLE IF NOT EXISTS "location" (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    seller_id INTEGER NOT NULL REFERENCES seller_profile(user_id),
    name VARCHAR(255) NOT NULL,
    type VARCHAR(20) DEFAULT 'WAREHOUSE',
    is_active BOOLEAN DEFAULT TRUE,
    priority INTEGER DEFAULT 0,
    address_id INTEGER NOT NULL REFERENCES address(id),
    CONSTRAINT unique_location_name_per_seller UNIQUE (seller_id, name)
);

CREATE INDEX idx_location_seller_id ON location(seller_id);

-- Create inventory table
CREATE TABLE IF NOT EXISTS inventory (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    variant_id INTEGER NOT NULL,
    location_id INTEGER NOT NULL REFERENCES location(id),
    quantity INTEGER DEFAULT 0,
    reserved_quantity INTEGER DEFAULT 0,
    threshold INTEGER DEFAULT 0,
    bin_location VARCHAR(50)
);

CREATE UNIQUE INDEX idx_inv_var_loc ON inventory(variant_id, location_id);

-- Create inventory_transaction table
CREATE TABLE IF NOT EXISTS inventory_transaction (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    inventory_id INTEGER NOT NULL REFERENCES inventory(id),
    type VARCHAR(20) NOT NULL,
    quantity INTEGER NOT NULL,
    before_quantity INTEGER NOT NULL,
    after_quantity INTEGER NOT NULL,
    performed_by INTEGER NOT NULL,
    reference_id VARCHAR(255),
    reference_type VARCHAR(50),
    reason TEXT NOT NULL,
    note TEXT
);

CREATE INDEX idx_inv_txn_inventory_id ON inventory_transaction(inventory_id);
CREATE INDEX idx_inv_txn_reference ON inventory_transaction(reference_id, reference_type);

-- Create inventory_reservation table
CREATE TABLE IF NOT EXISTS inventory_reservation (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    inventory_id INTEGER NOT NULL REFERENCES inventory(id),
    reference_id VARCHAR(255) NOT NULL,
    quantity INTEGER NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    status VARCHAR(20) DEFAULT 'PENDING'
);

CREATE INDEX idx_inv_res_inventory_id ON inventory_reservation(inventory_id);
CREATE INDEX idx_inv_res_reference_id ON inventory_reservation(reference_id);
CREATE INDEX idx_inv_res_expires_at ON inventory_reservation(expires_at);

-- Create stock_transfer table
CREATE TABLE IF NOT EXISTS stock_transfer (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    reference_number VARCHAR(255) UNIQUE,
    from_location_id INTEGER NOT NULL REFERENCES location(id),
    to_location_id INTEGER NOT NULL REFERENCES location(id),
    status VARCHAR(20) DEFAULT 'PENDING',
    requested_by INTEGER,
    shipped_at TIMESTAMPTZ,
    received_at TIMESTAMPTZ
);

-- Create stock_transfer_items table
CREATE TABLE IF NOT EXISTS stock_transfer_item (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    stock_transfer_id INTEGER NOT NULL REFERENCES stock_transfer(id),
    variant_id INTEGER NOT NULL,
    quantity INTEGER NOT NULL
);

CREATE INDEX idx_st_items_transfer_id ON stock_transfer_item(stock_transfer_id);
