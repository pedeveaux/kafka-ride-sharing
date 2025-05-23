FROM debian:bookworm-slim
WORKDIR /app

# Install librdkafka runtime
RUN apt-get update && apt-get install -y librdkafka1 && rm -rf /var/lib/apt/lists/*

COPY /bin/producer .
ENTRYPOINT ["/app/producer"]