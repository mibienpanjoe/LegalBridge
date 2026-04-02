FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/server  ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/migrate ./cmd/migrate

# alpine gives us a shell (needed by Render's preDeployCommand) while
# staying minimal (~10 MB). The Go binaries are fully static (CGO_ENABLED=0)
# so no libc is required.
FROM alpine:3

RUN addgroup -S app && adduser -S app -G app
COPY --from=builder /bin/server  /server
COPY --from=builder /bin/migrate /migrate

USER app
EXPOSE 8080

ENTRYPOINT ["/server"]
