FROM golang:1.22-alpine AS build

WORKDIR /src
ENV GOPROXY=https://goproxy.cn,direct
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/server ./cmd/server

FROM alpine:3.20

RUN adduser -D -H app
WORKDIR /app
COPY --from=build /out/server /app/server
USER app

EXPOSE 8080
ENTRYPOINT ["/app/server"]
