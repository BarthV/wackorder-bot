# Build stage
FROM golang:1.26-bookworm AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /wackorder ./cmd/wackorder

# Runtime stage — minimal distroless image, no shell
FROM gcr.io/distroless/static-debian12:nonroot

# nonroot user (UID 65532) — explicit numeric ID for Kubernetes securityContext
USER 65532:65532

COPY --from=build --chown=65532:65532 /wackorder /wackorder

# DB_PATH default; mount a PVC at /data in your pod spec for persistence
ENV DB_PATH=/data/wackorder.db

ENTRYPOINT ["/wackorder"]
