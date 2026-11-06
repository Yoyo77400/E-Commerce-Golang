FROM golang:1.27-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/ ./cmd/...

FROM alpine:3.22

RUN adduser -D -u 1000 shop

COPY --from=build /out/server /out/client /out/admin /out/sshd /usr/local/bin/

RUN mkdir -p /home/shop/.ssh && chown -R shop:shop /home/shop

USER shop
WORKDIR /home/shop

EXPOSE 8080 2222 2223

CMD ["server"]