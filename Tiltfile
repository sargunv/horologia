POSTGRES_PORT = 5432
OIDC_PORT = 5556
SERVER_PORT = 8080
WEB_PORT = 5173

common_env = {
    "TEND_DB": "postgres://postgres:postgres@localhost:%d/tend?sslmode=disable" % POSTGRES_PORT,
    "TEND_LOG_FORMAT": "text",
    "TEND_LOG_LEVEL": "debug",
    "TEND_OIDC_ISSUER": "http://localhost:%d/" % OIDC_PORT,
    "TEND_OIDC_CLIENT_ID": "tend",
    "TEND_OIDC_CLIENT_SECRET": "tend-dev-secret",
    "TEND_OIDC_REDIRECT_URL": "http://localhost:%d/api/auth/oidc/callback" % WEB_PORT,
    "TEND_SECURE_COOKIES": "false",
}

docker_compose("docker-compose.yml")
dc_resource("postgres", labels=["infra"])

local_resource(
    "oidc",
    serve_cmd="mise run //server:run-dev-oidc",
    serve_env=common_env,
    deps=["server"],
    readiness_probe=probe(
        http_get=http_get_action(port=OIDC_PORT, path="/.well-known/openid-configuration"),
        initial_delay_secs=2,
        period_secs=5,
        failure_threshold=10,
    ),
    links=["http://localhost:%d/" % OIDC_PORT],
    labels=["infra"],
)

local_resource(
    "server",
    serve_cmd="mise run //server:run",
    serve_env=common_env,
    deps=["server"],
    resource_deps=["oidc", "postgres"],
    readiness_probe=probe(
        exec=exec_action(["nc", "-z", "localhost", str(SERVER_PORT)]),
        initial_delay_secs=2,
        period_secs=5,
        failure_threshold=10,
    ),
    links=["http://localhost:%d/" % SERVER_PORT],
    labels=["app"],
)

local_resource(
    "seed",
    cmd="mise run //server:seed",
    env=common_env,
    resource_deps=["server"],
    labels=["infra"],
)

local_resource(
    "web",
    serve_cmd="mise run //web:dev",
    resource_deps=["server"],
    links=["http://localhost:%d/" % WEB_PORT],
    labels=["app"],
)
