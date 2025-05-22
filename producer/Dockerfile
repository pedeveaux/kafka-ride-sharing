FROM gcr.io/distroless/static:nonroot
WORKDIR /app
COPY /bin/producer .
USER nonroot
ENTRYPOINT ["/app/producer"]