FROM node:24-alpine AS web
WORKDIR /app
COPY package.json package-lock.json ./
COPY frontend/package.json frontend/package.json
RUN npm ci
COPY frontend/ frontend/
RUN npm run build

FROM golang:1.27-alpine AS api
ARG BUILD_VERSION=local
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ .
RUN go test ./... && go vet ./... && CGO_ENABLED=0 go build -trimpath -ldflags "-X main.buildVersion=${BUILD_VERSION}" -o /server ./cmd/server
RUN CGO_ENABLED=0 go build -trimpath -o /migrate-legacy ./cmd/migrate-legacy && CGO_ENABLED=0 go build -trimpath -o /provision-accounts ./cmd/provision-accounts

FROM alpine:3.23
ARG BUILD_VERSION=local
LABEL org.opencontainers.image.revision=$BUILD_VERSION
RUN apk add --no-cache ca-certificates && addgroup -g 10001 app && adduser -D -u 10001 -G app app
WORKDIR /app
COPY --from=api /server /app/server
COPY --from=api /migrate-legacy /app/migrate-legacy
COPY --from=api /provision-accounts /app/provision-accounts
COPY --from=web /app/frontend/dist /app/frontend/dist
RUN mkdir -p /app/data/runtime /app/data/evidence /app/data/credentials && chown -R app:app /app/data && chmod 700 /app/data/evidence /app/data/credentials
USER app
ENV HOST=0.0.0.0 PORT=8080
EXPOSE 8080
CMD ["/app/server"]
