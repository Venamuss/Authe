FROM golang:1.26.5 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/migrator ./cmd/migrator/main.go
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/user ./cmd/user/main.go

# migrations
FROM alpine:3.24 AS migrator 
WORKDIR /app
COPY --from=builder /bin/migrator /bin/migrator
CMD ["/bin/migrator"]

# user service
FROM alpine:3.24 AS user
WORKDIR /app
COPY --from=builder /bin/user /bin/user
EXPOSE 8080
CMD ["/bin/user"]
