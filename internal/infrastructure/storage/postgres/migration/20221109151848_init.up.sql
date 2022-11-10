CREATE TABLE price
(
    asset_hash varchar(32) NOT NULL PRIMARY KEY,
    amount decimal NOT NULL,
    basis_point integer NOT NULL
);

CREATE TABLE topup
(
    id            varchar(36) NOT NULL PRIMARY KEY,
    status        integer     NOT NULL,
    partial_tx    text,
    asset_hash    varchar(32) NOT NULL,
    asset_amount  integer     NOT NULL,
    asset_price   decimal     NOT NULL,
    fee_amount    integer     NOT NULL,
    requested_at  timestamp   NOT NULL,
    completed_at  timestamp,
    settled_at    timestamp,
    expire_at     timestamp,
    spread_amount integer     NOT NULL
);

CREATE TABLE whitelist
(
    script varchar(10000) NOT NULL PRIMARY KEY
);