OIDC_PORT = int(os.environ["OIDC_PORT"])
SERVER_PORT = int(os.environ["SERVER_PORT"])
WEB_PORT = int(os.environ["WEB_PORT"])

horologia_db = os.environ.get("HOROLOGIA_DB", "")
manage_postgres = horologia_db == ""

if manage_postgres:
    POSTGRES_PORT = int(os.environ["POSTGRES_PORT"])
    horologia_db = "postgres://postgres:postgres@localhost:%d/horologia?sslmode=disable" % POSTGRES_PORT

common_env = {
    "HOROLOGIA_ADDR": ":%d" % SERVER_PORT,
    "HOROLOGIA_DB": horologia_db,
    "HOROLOGIA_PUBLIC_URL": "http://localhost:%d" % SERVER_PORT,
    "HOROLOGIA_LOG_FORMAT": "text",
    "HOROLOGIA_LOG_LEVEL": "debug",
    "HOROLOGIA_OIDC_ISSUER": "http://localhost:%d/" % OIDC_PORT,
    "HOROLOGIA_OIDC_CLIENT_ID": "horologia",
    "HOROLOGIA_OIDC_CLIENT_SECRET": "horologia-dev-secret",
    "HOROLOGIA_SECURE_COOKIES": "false",
    "HOROLOGIA_HIBP_ENABLED": "false",
    "HOROLOGIA_INIT_OWNER_EMAIL": "admin@localhost",
    "HOROLOGIA_INIT_OWNER_NAME": "Admin",
    "HOROLOGIA_INIT_OWNER_PASSWORD": "password",
}

if manage_postgres:
    local_resource(
        "postgres",
        serve_cmd="""bash -c '
if [ ! -f "$PGDATA/PG_VERSION" ]; then
  initdb -D "$PGDATA" -U postgres --auth=trust --no-locale -E UTF8;
fi &&
exec postgres -D "$PGDATA" -p $POSTGRES_PORT
'""",
        serve_env={
            "PGDATA": os.environ["PGDATA"],
            "POSTGRES_PORT": str(POSTGRES_PORT),
        },
        readiness_probe=probe(
            exec=exec_action(["pg_isready", "-h", "localhost", "-p", str(POSTGRES_PORT), "-U", "postgres"]),
            initial_delay_secs=3,
            period_secs=2,
            failure_threshold=10,
        ),
        labels=["infra"],
    )

    local_resource(
        "createdb",
        cmd="createdb -h localhost -p %d -U postgres horologia 2>/dev/null || true" % POSTGRES_PORT,
        resource_deps=["postgres"],
        labels=["infra"],
    )

local_resource(
    "oidc",
    cmd="mise run //server:build-dev-oidc",
    serve_cmd="server/dev-oidc",
    serve_env=dict(common_env, **{
        "DEV_OIDC_PORT": str(OIDC_PORT),
        "DEV_OIDC_CALLBACK_URL": "http://localhost:%d/app/auth/oidc/callback" % SERVER_PORT,
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
    server_resource_deps.append("createdb")

local_resource(
    "server",
    cmd="mise run //server:build",
    serve_cmd="server/horologia-server serve",
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
    "web",
    cmd="mise run //clients/web:generate",
    serve_cmd="cd clients/web && pnpm exec vp dev",
    serve_env={
        "SERVER_PORT": str(SERVER_PORT),
        "WEB_PORT": str(WEB_PORT),
    },
    resource_deps=["server"],
    links=["http://localhost:%d/" % WEB_PORT],
    labels=["app"],
)
