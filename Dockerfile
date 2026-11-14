# SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
#
# SPDX-License-Identifier: Unlicense

FROM --platform=$BUILDPLATFORM golang:1.26.11-bookworm@sha256:dc40a13b61304647ed7e5d581d246b86413dfc29d63c6d018310feb93cd98fc0 AS build

ENV CGO_ENABLED=0 GOTOOLCHAIN=local
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .

ARG TARGETOS
ARG TARGETARCH
RUN export GOOS="$TARGETOS" GOARCH="$TARGETARCH" && \
    go build -trimpath -o /build/server ./cmd/server && \
    go build -trimpath -o /build/insert-user ./cmd/insert-user

FROM cloudron/base:6.0.1@sha256:7b1e729960c3ca312d26e886917ba89483241bba0219ff292858e10d9d5768ee

WORKDIR /app/code

COPY --from=build /build/server /app/code/server
COPY --from=build /build/insert-user /app/code/insert-user
COPY --chmod=0755 start.sh room-admin /app/code/

EXPOSE 3000 8008

CMD ["/app/code/start.sh"]
