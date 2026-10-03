# --- build stage ---
FROM golang:1.22-alpine AS build
WORKDIR /app

COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/server ./cmd/server

# --- run stage ---
FROM alpine:3.19
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /bin/server /app/server
COPY migrations /app/migrations

EXPOSE 8080
ENTRYPOINT ["/app/server"]
