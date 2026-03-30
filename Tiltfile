OIDC_PORT = int(os.environ["OIDC_PORT"])
SERVER_PORT = int(os.environ["SERVER_PORT"])
WEB_PORT = int(os.environ["WEB_PORT"])

tend_db = os.environ.get("TEND_DB", "")
manage_postgres = tend_db == ""

if manage_postgres:
    POSTGRES_PORT = int(os.environ["POSTGRES_PORT"])
    tend_db = "postgres://postgres:postgres@localhost:%d/tend?sslmode=disable" % POSTGRES_PORT

common_env = {
    "TEND_ADDR": ":%d" % SERVER_PORT,
    "TEND_DB": tend_db,
    "TEND_LOG_FORMAT": "text",
    "TEND_LOG_LEVEL": "debug",
    "TEND_OIDC_ISSUER": "http://localhost:%d/" % OIDC_PORT,
    "TEND_OIDC_CLIENT_ID": "tend",
    "TEND_OIDC_CLIENT_SECRET": "tend-dev-secret",
    "TEND_OIDC_REDIRECT_URL": "http://localhost:%d/api/auth/oidc/callback" % WEB_PORT,
    "TEND_SECURE_COOKIES": "false",
    "TEND_HIBP_ENABLED": "false",
}

if manage_postgres:
    docker_compose("docker-compose.yml")
    dc_resource("postgres", labels=["infra"])

local_resource(
    "oidc",
    serve_cmd="mise run //server:run-dev-oidc",
    serve_env=dict(common_env, **{
        "DEV_OIDC_PORT": str(OIDC_PORT),
        "DEV_OIDC_CALLBACK_URL": "http://localhost:%d/api/auth/oidc/callback" % WEB_PORT,
    }),
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

server_resource_deps = ["oidc"]
if manage_postgres:
    server_resource_deps.append("postgres")

local_resource(
    "server",
    serve_cmd="mise run //server:run",
    serve_env=common_env,
    deps=["server"],
    resource_deps=server_resource_deps,
    readiness_probe=probe(
        http_get=http_get_action(port=SERVER_PORT, path="/healthz"),
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
    serve_env={
        "SERVER_PORT": str(SERVER_PORT),
        "WEB_PORT": str(WEB_PORT),
    },
    resource_deps=["server"],
    links=["http://localhost:%d/" % WEB_PORT],
    labels=["app"],
)
