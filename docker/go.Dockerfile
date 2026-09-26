# Go build of Pinchflat. Replaces selfhosted.Dockerfile at cutover (W5).
# Runtime layer is unchanged: same tools, volumes, env, port and healthcheck.
ARG GO_VERSION=1.25
ARG DEBIAN_VERSION=bookworm-20250428-slim
ARG RUNNER_IMAGE="debian:${DEBIAN_VERSION}"

FROM golang:${GO_VERSION}-bookworm AS builder

ARG TARGETPLATFORM
ARG VERSION=dev
RUN echo "Building for ${TARGETPLATFORM:?}"

# FFmpeg: yt-dlp's recommended build, always the current one.
RUN export FFMPEG_DOWNLOAD=$(case ${TARGETPLATFORM:-linux/amd64} in \
    "linux/amd64")   echo "https://github.com/yt-dlp/FFmpeg-Builds/releases/download/latest/ffmpeg-master-latest-linux64-gpl.tar.xz"   ;; \
    "linux/arm64")   echo "https://github.com/yt-dlp/FFmpeg-Builds/releases/download/latest/ffmpeg-master-latest-linuxarm64-gpl.tar.xz" ;; \
    *)               echo ""        ;; esac) && \
    curl -fL ${FFMPEG_DOWNLOAD} --output /tmp/ffmpeg.tar.xz && \
    tar -xf /tmp/ffmpeg.tar.xz --strip-components=2 --no-anchored -C /usr/local/bin/ "ffmpeg" && \
    tar -xf /tmp/ffmpeg.tar.xz --strip-components=2 --no-anchored -C /usr/local/bin/ "ffprobe"

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd cmd
COPY internal internal
# Templates (_templ.go) and CSS/JS (internal/web/static/assets) are generated
# and committed, so no Node, templ or Tailwind is needed here.
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /out/pinchflat ./cmd/pinchflat

## -- Release Stage --

FROM ${RUNNER_IMAGE}

ARG TARGETPLATFORM
ARG PORT=8945

COPY --from=builder /usr/local/bin/ffmpeg /usr/bin/ffmpeg
COPY --from=builder /usr/local/bin/ffprobe /usr/bin/ffprobe

RUN apt-get update -y && \
    apt-get install -y \
      locales \
      ca-certificates \
      tzdata \
      python3-mutagen \
      curl \
      zip \
      openssh-client \
      nano \
      python3 \
      pipx \
      jq \
      # unzip is needed for Deno
      unzip \
      procps && \
    # Install Deno - required for YouTube downloads (See yt-dlp#14404)
    curl -fsSL https://deno.land/install.sh | DENO_INSTALL=/usr/local sh -s -- -y --no-modify-path && \
    # Apprise
    export PIPX_HOME=/opt/pipx && \
    export PIPX_BIN_DIR=/usr/local/bin && \
    pipx install apprise && \
    # yt-dlp
    export YT_DLP_DOWNLOAD=$(case ${TARGETPLATFORM:-linux/amd64} in \
    "linux/amd64")   echo "https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp_linux"   ;; \
    "linux/arm64")   echo "https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp_linux_aarch64" ;; \
    *)               echo "https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp_linux"        ;; esac) && \
    curl -L ${YT_DLP_DOWNLOAD} -o /usr/local/bin/yt-dlp && \
    chmod a+rx /usr/local/bin/yt-dlp && \
    yt-dlp -U && \
    sed -i '/en_US.UTF-8/s/^# //g' /etc/locale.gen && locale-gen && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

ENV LANG=en_US.UTF-8
ENV LANGUAGE=en_US:en
ENV LC_ALL=en_US.UTF-8

WORKDIR "/app"

RUN mkdir -p /config /downloads /etc/yt-dlp/plugins /app/bin && \
  chmod ugo+rw /etc/yt-dlp /etc/yt-dlp/plugins /usr/local/bin /usr/local/bin/yt-dlp

ENV PORT=${PORT}
ENV RUN_CONTEXT="selfhosted"
ENV UMASK=022
EXPOSE ${PORT}

COPY --from=builder /out/pinchflat /app/bin/pinchflat
# Old entrypoints kept for compose files that reference them.
RUN printf '#!/bin/sh\nexec /app/bin/pinchflat start "$@"\n' > /app/bin/docker_start && \
    printf '#!/bin/sh\nexec /app/bin/pinchflat migrate "$@"\n' > /app/bin/migrate && \
    chmod a+rx /app/bin/docker_start /app/bin/migrate

HEALTHCHECK --interval=30s --start-period=15s \
  CMD curl --fail http://localhost:${PORT}/healthcheck || exit 1

CMD ["/app/bin/pinchflat"]
