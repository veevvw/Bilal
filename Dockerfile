FROM golang:1.27 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go generate

RUN CGO_ENABLED=0 GOOS=linux go build -o tgweb main.go

FROM debian:bookworm-slim

WORKDIR /app

RUN apt-get update && apt-get install -y \
    ca-certificates \
    ffmpeg \
    python3 \
    python3-venv \
    libstdc++6 \
    curl \
    wget \
    unzip \
    && rm -rf /var/lib/apt/lists/*


RUN wget -O /usr/local/bin/yt-dlp \
    https://github.com/yt-dlp/yt-dlp-nightly-builds/releases/latest/download/yt-dlp_linux \
    && chmod +x /usr/local/bin/yt-dlp

RUN curl -fsSL https://deno.land/install.sh | sh \
    && ln -sf /root/.deno/bin/deno /usr/local/bin/deno

ENV DENO_INSTALL="/root/.deno"
ENV PATH="${DENO_INSTALL}/bin:${PATH}"

COPY voicechat/requirements.txt /app/voicechat/requirements.txt
RUN python3 -m venv /opt/voice-assistant \
    && /opt/voice-assistant/bin/pip install --no-cache-dir -r /app/voicechat/requirements.txt
ENV VOICE_CHAT_PYTHON="/opt/voice-assistant/bin/python"
COPY voicechat /app/voicechat

COPY --from=builder /app/tgweb .
COPY --from=builder /app/libtdjson.so.* ./

CMD ["./tgweb"]
