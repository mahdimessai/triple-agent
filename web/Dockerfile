# syntax=docker/dockerfile:1

FROM oven/bun:1-alpine AS deps

WORKDIR /app
COPY package.json bun.lock* ./
COPY web/package.json ./web/package.json
RUN bun install --frozen-lockfile

FROM oven/bun:1-alpine AS builder

WORKDIR /app
ENV NEXT_TELEMETRY_DISABLED=1

COPY --from=deps /app/node_modules ./node_modules
COPY package.json bun.lock* ./
COPY web ./web

ARG NEXT_PUBLIC_TRIPLE_AGENT_API_URL=http://localhost:8080
ENV NEXT_PUBLIC_TRIPLE_AGENT_API_URL=$NEXT_PUBLIC_TRIPLE_AGENT_API_URL

RUN bun run --filter triple-agent-web build

FROM oven/bun:1-alpine AS runner

WORKDIR /app
ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1
ENV PORT=3000

RUN addgroup --system --gid 1001 bunjs \
    && adduser --system --uid 1001 -G bunjs bunjs

COPY --from=builder --chown=bunjs:bunjs /app/web/.next/standalone ./
COPY --from=builder /app/web/public ./web/public
COPY --from=builder --chown=bunjs:bunjs /app/web/.next/static ./web/.next/static

USER bunjs
EXPOSE 3000

CMD ["bun", "web/server.js"]
