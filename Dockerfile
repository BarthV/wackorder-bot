# Build stage
FROM golang:1.26-bookworm AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /wackorder ./cmd/wackorder

# Runtime stage — minimal distroless image, no shell
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /wackorder /wackorder

# SQLite database lives here; mount a named volume for persistence
VOLUME /data
ENV DB_PATH=/data/wackorder.db

ENTRYPOINT ["/wackorder"]
