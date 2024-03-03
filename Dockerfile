# Stage 1: Build application (with dependencies)
FROM golang:alpine AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o /app/echo .

# Stage 2: Slim image (application binary only)
FROM alpine AS final

WORKDIR /app

COPY --from=build /app/echo /app/echo

EXPOSE 8080

CMD ["/app/echo"]
