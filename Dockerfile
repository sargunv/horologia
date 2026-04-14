# syntax=docker/dockerfile:1

FROM gcr.io/distroless/static-debian12:nonroot

ARG TARGETPLATFORM

COPY ${TARGETPLATFORM}/horologia-server /horologia-server

HEALTHCHECK --start-period=15s --interval=10s --timeout=5s --retries=3 \
    CMD ["/horologia-server", "healthcheck"]

ENTRYPOINT ["/horologia-server"]
CMD ["serve"]
