# SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
#
# SPDX-License-Identifier: Unlicense

# Builder
FROM golang:1.22-bookworm AS build

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
      build-essential \
      ca-certificates \
      git \
      gosu \
      libsqlite3-dev \
      pkg-config && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=1 go build -o /build/server ./cmd/server && \
    CGO_ENABLED=1 go build -o /build/insert-user ./cmd/insert-user

# Runtime
FROM cloudron/base:5.0.0

RUN mkdir -p /app/code /app/data && \
    apt-get update && \
    apt-get install -y --no-install-recommends \
      libsqlite3-0 && \
    rm -rf /var/lib/apt/lists/* && \
    chown -R cloudron:cloudron /app

WORKDIR /app/code

COPY --from=build /build/server /app/code/server
COPY --from=build /build/insert-user /app/code/insert-user
COPY start.sh /app/code/start.sh

RUN chmod +x /app/code/start.sh

USER root

EXPOSE 3000
EXPOSE 8008

CMD ["/app/code/start.sh"]
