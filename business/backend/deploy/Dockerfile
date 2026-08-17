# syntax=docker/dockerfile:1.7
FROM golang:1.25-alpine AS build
WORKDIR /workspace
COPY --from=kernel . ./kernel-go
COPY --from=protocol . ./protocol
COPY . ./backend
WORKDIR /workspace/backend
RUN go mod edit \
	-replace=github.com/lvtuopen-ai/kernel-go=/workspace/kernel-go \
	-replace=github.com/liveshop-platform/contracts=/workspace/protocol && \
	CGO_ENABLED=0 go build -trimpath -o /out/platform ./internal/cmd && \
	CGO_ENABLED=0 go build -trimpath -o /out/grpccerts ./cmd/grpccerts && \
	CGO_ENABLED=0 go build -trimpath -o /out/authorizationexport ./cmd/authorizationexport
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/platform /app/platform
COPY --from=build /out/grpccerts /app/grpccerts
COPY --from=build /out/authorizationexport /app/authorizationexport
COPY --from=build /workspace/backend/configs /app/configs
WORKDIR /app
USER nonroot:nonroot
ENTRYPOINT ["/app/platform"]
