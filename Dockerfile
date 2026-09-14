FROM golang:1.26-alpine AS build

WORKDIR /src

COPY go.mod go.sum* ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/ojirun ./cmd/bot

FROM alpine:3.22

RUN apk add --no-cache tzdata ca-certificates curl
RUN addgroup -S app && adduser -S app -G app

WORKDIR /app

COPY --from=build /out/ojirun /app/ojirun
COPY prompts /app/prompts
COPY migrations /app/migrations

RUN mkdir -p /app/storage/photos && chown -R app:app /app

USER app

ENTRYPOINT ["/app/ojirun"]
