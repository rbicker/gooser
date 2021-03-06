# builder
FROM golang:alpine AS builder
RUN apk update && apk add --no-cache git tzdata ca-certificates
ENV USER=app
ENV UID=10001
RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/nonexistent" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid "${UID}" \
    "${USER}"

WORKDIR $GOPATH/src/github.com/rbicker/gooser
COPY . .
RUN go mod download
RUN go mod verify
RUN CGO_ENABLED=0 go test ./...
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /go/bin/gooser-server ./cmd/gooser-server

# ---

# app image
FROM scratch
ENV PORT=50051
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group
# for working with timezones
COPY --from=builder /usr/share/zoneinfo  /usr/share/zoneinfo
# to have valid ca certs
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /go/bin/gooser-server /gooser-server
USER app:app
# EXPOSE 50051
ENTRYPOINT ["/gooser-server"]