FROM gcr.io/distroless/static:nonroot
WORKDIR /app
COPY ../bin/consumer .
USER nonroot
ENTRYPOINT ["/app/consumer"]