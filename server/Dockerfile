FROM golang:1.27-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY server ./server
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/triple-agent-server ./server/cmd/tripleagent

FROM alpine:3.22

RUN addgroup -S app && adduser -S -G app app
COPY --from=build /out/triple-agent-server /usr/local/bin/triple-agent-server
USER app

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/triple-agent-server"]
