# ContextRail — Cloud Run baseline image
#
# Builds the React workspace, compiles the Go read-only registry service and
# packages both with the two governed Project fixtures. The resulting image is
# read-only at runtime: GET-only API, no persistence, no cloud credentials.
#
# Stage 1: workspace bundle
FROM node:22-alpine AS workspace
WORKDIR /src/frontend
ENV PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY frontend/ ./
RUN npm run build

# Stage 2: Go service
FROM golang:1.24-alpine AS service
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ ./cmd/
COPY internal/ ./internal/
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/context-rail ./cmd/context-rail

# Stage 3: minimal runtime
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=service /out/context-rail /app/context-rail
COPY --from=workspace /src/frontend/dist /app/static
# Governed Project fixtures. These are explicit roots, not a filesystem scan.
COPY demo/order-operations-portal /app/fixtures/order-operations-portal
COPY examples/support-insights /app/fixtures/support-insights

ENV CONTEXT_RAIL_STATIC_DIR=/app/static \
    CONTEXT_RAIL_FIXTURE_ROOTS=/app/fixtures/order-operations-portal,/app/fixtures/support-insights \
    PORT=8080
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/context-rail"]
