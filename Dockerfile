FROM golang:1.27-alpine

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o api ./cmd/api

ENTRYPOINT ["/app/api"]