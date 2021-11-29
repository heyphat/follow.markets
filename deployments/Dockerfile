# Compile
FROM golang:1.17 as builder

WORKDIR /follow.market

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build ./cmd/app

# This image is to deploy the binary
FROM alpine
RUN apk update
RUN apk add --update ca-certificates tzdata
COPY --from=builder ./follow.market/app app
COPY configs/configs.json configs/configs.json
CMD ["./app"]
