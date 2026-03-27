FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git build-base oniguruma-dev

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 go build -o /archives ./cmd/server/

FROM alpine:3.21

RUN apk add --no-cache \
    ca-certificates \
    file \
    nodejs \
    npm \
    oniguruma \
 && npm install -g repomix

WORKDIR /app
COPY --from=builder /archives /app/archives
COPY templates/ /app/templates/
COPY static/ /app/static/
COPY openapi/ /app/openapi/

EXPOSE 5000
ENV PORT=5000

CMD ["/app/archives"]
