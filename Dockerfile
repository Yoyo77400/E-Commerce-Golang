FROM golang:1.27-alpine AS build

WORKDIR /src

COPY go.mod go.sum* ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/client ./cmd/client
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/admin ./cmd/admin

FROM alpine:3.22

RUN adduser -D -u 1000 shop

COPY --from=build /out/server /usr/local/bin/server
COPY --from=build /out/client /usr/local/bin/client
COPY --from=build /out/admin /usr/local/bin/admin

USER shop
WORKDIR /home/shop

EXPOSE 8080

CMD ["server"]
