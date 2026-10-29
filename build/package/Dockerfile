FROM golang:alpine AS builder

WORKDIR /app

COPY go.* ./
RUN go mod download

COPY . ./

# CGO_ENABLED=0 is required so it runs inside scratch
RUN CGO_ENABLED=0 go build -o /app/server ./cmd/httpserver

FROM scratch

COPY --from=builder /app/server /server

CMD ["/server"]