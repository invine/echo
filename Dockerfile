# Stage 1: Build application (with dependencies)
FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.22.0 AS build

ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o /app/echo ./internal/cmd/

# Stage 2: Slim image (application binary only)
FROM --platform=${TARGETPLATFORM:-linux/amd64} scratch AS final

WORKDIR /app

COPY --from=build /app/echo /app/echo

EXPOSE 8080

CMD ["/app/echo"]
