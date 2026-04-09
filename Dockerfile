FROM golang:1.26-alpine AS build
WORKDIR /app
COPY go.mod .
COPY . .
RUN go build -ldflags="-s -w" -o server ./cmd/server

FROM alpine:3.21
WORKDIR /app
COPY --from=build /app/server .
ENV ADDR=:8080
ENV PACK_FILE=/app/data/packs.json
ENTRYPOINT ["./server"]
