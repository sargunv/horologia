-- +goose Up

INSERT INTO oauth_clients (
    client_id,
    display_name,
    redirect_uris,
    loopback_redirects,
    client_secret_hash,
    is_first_party,
    created_at
)
VALUES (
    'horologia-mobile',
    'Horologia',
    '{horologia://oauth}',
    TRUE,
    NULL,
    TRUE,
    now()
)
ON CONFLICT (client_id) DO NOTHING;

-- +goose Down

DELETE FROM oauth_clients
WHERE client_id = 'horologia-mobile';
