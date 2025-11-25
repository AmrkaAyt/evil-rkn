# === build stage ===
FROM golang:1.24.7 AS builder


WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

WORKDIR /app/cmd/rkn-service
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/rkn-service .

# === runtime stage ===
FROM gcr.io/distroless/base-debian12

WORKDIR /app

COPY --from=builder /app/rkn-service .

# Создаём непривилегированного пользователя (в distroless уже есть nonroot)
USER nonroot:nonroot

EXPOSE 80 9090

ENV HTTP_ADDR=:80
ENV GRPC_ADDR=:9090
ENV RKN_API_BASE_URL=https://reestr.rublacklist.net/api/v3
ENV UPDATE_INTERVAL=6h

ENTRYPOINT ["/app/rkn-service"]
