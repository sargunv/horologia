# syntax=docker/dockerfile:1

ARG NODE_VERSION
ARG GO_VERSION

# --- Stage 1: Install Node dependencies ---
# Uses the workspace root lockfile so both api/ and web/ packages are covered.
# Layer cache busts only when lockfile or package.json files change.
FROM node:${NODE_VERSION}-bookworm-slim AS node-deps
RUN npm install -g --force corepack && corepack enable pnpm
WORKDIR /src
COPY pnpm-workspace.yaml pnpm-lock.yaml package.json ./
COPY api/package.json ./api/
COPY api/emitters/typespec-mcp-go/package.json ./api/emitters/typespec-mcp-go/
COPY web/package.json ./web/
RUN pnpm install --frozen-lockfile

# --- Stage 2: Build TypeSpec API ---
FROM node-deps AS api-build
COPY api/ ./api/
RUN pnpm --filter @horologia/typespec-mcp-go run build
RUN cd api && pnpm exec tsp compile .

# --- Stage 3: Build React SPA ---
FROM api-build AS web-build
COPY web/ ./web/
RUN cd web \
    && pnpm exec openapi-typescript ../api/tsp-output/schema/openapi.yaml \
         -o src/api/schema.d.ts \
    && pnpm exec tsr generate \
    && pnpm exec vp build

# --- Stage 4: Download Go module dependencies ---
FROM golang:${GO_VERSION}-bookworm AS go-deps
COPY api/go.mod /src/api/go.mod
COPY api/go.sum /src/api/go.sum
WORKDIR /src/api
RUN go mod download

WORKDIR /src/server
COPY server/go.mod server/go.sum ./
RUN go mod download

# --- Stage 5: Build Go server binary ---
FROM go-deps AS server-build

# Copy api source needed by the shared generated module.
COPY api/ /src/api/

# Copy server source.
COPY server/ ./

# Copy the compiled openapi.yaml needed by codegen tools.
COPY --from=api-build /src/api/tsp-output/schema/openapi.yaml \
    /src/api/tsp-output/schema/openapi.yaml

# Run code generation (tools invoked via go run from module dependencies).
RUN cd /src/api \
 && go run github.com/ogen-go/ogen/cmd/ogen \
      --target gen \
      --package gen \
      --clean \
      tsp-output/schema/openapi.yaml \
 && cd /src/server \
 && go run github.com/sqlc-dev/sqlc/cmd/sqlc generate

# Place the SPA dist where the go:embed directive can find it.
COPY --from=web-build /src/web/dist/ ./internal/webui/dist/

# Build a fully static binary.
RUN CGO_ENABLED=0 go build \
      -trimpath \
      -ldflags="-s -w" \
      -o /horologia-server \
      ./cmd/server

# --- Stage 6: Minimal runtime image ---
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=server-build /horologia-server /horologia-server
HEALTHCHECK --start-period=15s --interval=10s --timeout=5s --retries=3 \
    CMD ["/horologia-server", "healthcheck"]
ENTRYPOINT ["/horologia-server"]
CMD ["serve"]
