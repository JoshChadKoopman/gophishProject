# Stage 1: Minify client side assets (JavaScript)
FROM node:18-alpine AS build-js

RUN npm install -g gulp gulp-cli

WORKDIR /build
COPY package.json yarn.lock .babelrc gulpfile.js webpack.config.js ./
COPY static/ ./static/
COPY templates/ ./templates/
RUN npm install --only=dev
RUN gulp


# Stage 2: Build Golang binary
FROM golang:1.21-bullseye AS build-golang

RUN apt-get update && apt-get install --no-install-recommends -y gcc libc6-dev git && rm -rf /var/lib/apt/lists/*

WORKDIR /go/src/github.com/gophish/gophish
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -v -o gophish .


# Stage 3: Runtime container
FROM debian:bookworm-slim

RUN useradd -m -d /opt/gophish -s /bin/bash app

RUN apt-get update && \
	apt-get install --no-install-recommends -y jq libcap2-bin ca-certificates && \
	apt-get clean && \
	rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

WORKDIR /opt/gophish
COPY --from=build-golang /go/src/github.com/gophish/gophish/gophish ./gophish
COPY --from=build-golang /go/src/github.com/gophish/gophish/VERSION ./VERSION
COPY --from=build-golang /go/src/github.com/gophish/gophish/config.json ./config.json
COPY --from=build-golang /go/src/github.com/gophish/gophish/db/ ./db/
COPY --from=build-golang /go/src/github.com/gophish/gophish/static/ ./static/
COPY --from=build-golang /go/src/github.com/gophish/gophish/templates/ ./templates/
# Overwrite with freshly built/minified JS and CSS (must come AFTER the full static copy)
COPY --from=build-js /build/static/js/dist/ ./static/js/dist/
COPY --from=build-js /build/static/css/dist/ ./static/css/dist/
COPY --from=build-golang /go/src/github.com/gophish/gophish/docker/ ./docker/

RUN chmod +x ./docker/run.sh
RUN chown app. config.json
RUN mkdir -p /opt/gophish/data && chown app. /opt/gophish/data
RUN mkdir -p /opt/gophish/static/training_uploads && chown app. /opt/gophish/static/training_uploads
RUN setcap 'cap_net_bind_service=+ep' /opt/gophish/gophish

USER app
RUN sed -i 's/127.0.0.1/0.0.0.0/g' config.json
RUN chmod 600 config.json
RUN touch config.json.tmp

EXPOSE 3333 80

CMD ["./docker/run.sh"]
