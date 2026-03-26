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
}

go_deps = [
    "server/go.mod",
    "server/go.sum",
    "server/tools.go",
    "server/internal",
    "server/api",
    "server/cmd",
]

docker_compose("docker-compose.yml")
dc_resource("postgres", labels=["infra"])

local_resource(
    "oidc",
    cmd="cd server && go build -o tmp/dev-oidc ./cmd/dev-oidc",
    serve_cmd="./server/tmp/dev-oidc",
    serve_env=common_env,
    deps=go_deps,
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
    cmd="cd server && go build -o tmp/tend-server ./cmd/server",
    serve_cmd="./server/tmp/tend-server serve",
    serve_env=common_env,
    deps=go_deps,
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
    cmd="./server/tmp/tend-server create-admin --email admin@localhost --name Admin --password password",
    env=common_env,
    resource_deps=["server"],
    labels=["infra"],
)

local_resource(
    "web",
    serve_cmd="pnpm exec vp dev",
    serve_dir="./web",
    resource_deps=["server"],
    links=["http://localhost:%d/" % WEB_PORT],
    labels=["app"],
)
