# syntax=docker/dockerfile:1.7
FROM golang:1.25-alpine AS build
WORKDIR /workspace
COPY --from=kernel . ./kernel-go
COPY . ./backend
WORKDIR /workspace/backend
RUN go mod edit -replace=github.com/lvtuopen-ai/kernel-go=/workspace/kernel-go && \
	CGO_ENABLED=0 go build -trimpath -o /out/platform ./internal/platform/cmd && \
	CGO_ENABLED=0 go build -trimpath -o /out/grpccerts ./cmd/grpccerts
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/platform /app/platform
COPY --from=build /out/grpccerts /app/grpccerts
COPY --from=build /workspace/backend/configs /app/configs
WORKDIR /app
USER nonroot:nonroot
ENTRYPOINT ["/app/platform"]
