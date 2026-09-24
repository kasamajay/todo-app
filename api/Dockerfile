# Production image: a statically linked api binary on a small Alpine base -
# no Go toolchain, no air, no source bind mount at runtime. See
# decisions/0013-production-mode-nginx-static-bundle.md.
# (Dev still uses Dockerfile.dev + air hot reload.)

FROM golang:1.22 AS build

WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

# Alpine rather than scratch: the Google OAuth code-exchange makes outbound
# HTTPS calls and needs a CA bundle.
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

COPY --from=build /out/api /usr/local/bin/api

WORKDIR /app

EXPOSE 8080

CMD ["api"]
