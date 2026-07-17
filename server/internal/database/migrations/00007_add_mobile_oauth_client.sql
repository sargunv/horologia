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
    ARRAY['horologia://oauth/callback'],
    FALSE,
    NULL,
    TRUE,
    now()
)
ON CONFLICT (client_id) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    redirect_uris = EXCLUDED.redirect_uris,
    loopback_redirects = EXCLUDED.loopback_redirects,
    client_secret_hash = EXCLUDED.client_secret_hash,
    is_first_party = EXCLUDED.is_first_party;

-- +goose Down

DELETE FROM oauth_clients
WHERE client_id = 'horologia-mobile';
