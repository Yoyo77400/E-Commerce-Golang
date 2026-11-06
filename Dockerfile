FROM golang:1.27-alpine AS build

WORKDIR /src

COPY go.mod go.sum* ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM alpine:3.22

RUN adduser -D -u 1000 shop

COPY --from=build /out/server /usr/local/bin/server

USER shop
WORKDIR /home/shop

EXPOSE 8080

CMD ["server"]
