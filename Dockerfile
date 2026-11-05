FROM golang:1.26.10-alpine@sha256:a2012b4a1d1f9f7e9f44e0d39b805a901e856b1dd80b4e6ee4338db547050308 AS dependencies

RUN apk add --no-cache git build-base libjpeg-turbo-dev libwebp-dev

WORKDIR /build
ENV GOTOOLCHAIN=local

# Copiar apenas arquivos de dependências primeiro para cachear o download
COPY go.mod go.sum ./

# whatsmeow agora vem do proxy oficial (go.mau.fi/whatsmeow, sem replace local) —
# não há mais submódulo whatsmeow-lib para copiar.
RUN go mod download

FROM dependencies AS verification
COPY . .
RUN go test ./... -count=1 && go vet ./...
RUN go test -race ./pkg/runtimestate ./pkg/events/websocket ./pkg/events/webhook ./pkg/instance/handler ./pkg/instance/repository ./pkg/whatsmeow/service ./pkg/instance/service ./pkg/chat/service ./pkg/sendMessage/service ./pkg/utils ./pkg/user/service ./pkg/user/handler -count=1
ENTRYPOINT ["go", "test"]
CMD ["./pkg/whatsmeow/service", "-run", "TestHandshakeVersion|TestAuthStore", "-count=1", "-v", "-timeout", "90s"]

FROM verification AS build
ARG VERSION=0.7.2-nc.6
RUN CGO_ENABLED=1 go build -ldflags "-X main.version=${VERSION}" -o server ./cmd/evolution-go
RUN printf '%s\n' "$VERSION" > VERSION

FROM alpine:3.23.7@sha256:ff7f0c727dd796aa36787bfcd9a64bf35a6ce72a73451b3359fde849b6703d63 AS final

# poppler-utils provides pdftoppm, used to rasterize PDF page 1 for /send/media document thumbnails
RUN apk add --no-cache tzdata ffmpeg libjpeg-turbo libwebp poppler-utils

WORKDIR /app

COPY --from=build /build/server .
COPY --from=build /build/manager/dist ./manager/dist
COPY --from=build /build/VERSION ./VERSION
COPY --from=build /build/LICENSE ./LICENSE
COPY --from=build /build/NOTICE ./NOTICE
COPY --from=build /build/TRADEMARKS.md ./TRADEMARKS.md
COPY --from=build /build/nandocosta/RELEASE.json ./nandocosta-release.json

ENV TZ=America/Sao_Paulo
LABEL org.opencontainers.image.title="Evolution GO - nandocosta" \
      org.opencontainers.image.vendor="Nando Costa" \
      org.opencontainers.image.source="https://github.com/evolution-foundation/evolution-go" \
      org.opencontainers.image.version="0.7.2-nc.6"

ENTRYPOINT ["/app/server"]
