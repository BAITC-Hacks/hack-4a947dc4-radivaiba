FROM node:24-alpine AS web
WORKDIR /app
COPY package.json package-lock.json ./
COPY frontend/package.json frontend/package.json
RUN npm ci
COPY frontend/ frontend/
RUN npm run build

FROM golang:1.27-alpine AS api
WORKDIR /src
COPY backend/ .
RUN go test ./... && go vet ./... && CGO_ENABLED=0 go build -trimpath -o /server ./cmd/server

FROM alpine:3.23
RUN apk add --no-cache ca-certificates && addgroup -g 10001 app && adduser -D -u 10001 -G app app
WORKDIR /app
COPY --from=api /server /app/server
COPY --from=web /app/frontend/dist /app/frontend/dist
RUN mkdir -p /app/data/runtime && chown -R app:app /app/data
USER app
ENV HOST=0.0.0.0 PORT=8080
EXPOSE 8080
CMD ["/app/server"]
