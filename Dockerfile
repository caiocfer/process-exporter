# syntax=docker/dockerfile:1

FROM golang:1.26-bookworm AS build
WORKDIR /src

# Cache module downloads.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build a static binary for the distroless/cc runtime.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/process-exporter ./cmd/exporter

FROM gcr.io/distroless/cc-debian12
WORKDIR /
COPY --from=build /out/process-exporter /process-exporter

EXPOSE 9835
ENTRYPOINT ["/process-exporter"]
