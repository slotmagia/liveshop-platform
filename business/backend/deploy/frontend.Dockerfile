# syntax=docker/dockerfile:1.7
FROM node:22-alpine AS build
WORKDIR /workspace
ARG WORKSPACE
ARG SOURCE_DIR
COPY package.json package-lock.json ./
COPY packages ./packages
COPY ui ./ui
COPY ${SOURCE_DIR} ./${SOURCE_DIR}
RUN npm ci
RUN npm run build:packages && npm run build --workspace="$WORKSPACE" && mkdir -p /out && cp -R "$SOURCE_DIR/dist/." /out/

FROM nginxinc/nginx-unprivileged:1.27-alpine
COPY backend/deploy/nginx.frontend.conf /etc/nginx/conf.d/default.conf
COPY --from=build /out /usr/share/nginx/html
EXPOSE 8080
