# syntax=docker/dockerfile:1

FROM oven/bun:1-alpine AS builder

WORKDIR /app
ENV NEXT_TELEMETRY_DISABLED=1

COPY web/package.json web/bun.lock* ./
RUN bun install --frozen-lockfile

COPY web/ .

ARG NEXT_PUBLIC_TRIPLE_AGENT_API_URL=http://localhost:8080
ENV NEXT_PUBLIC_TRIPLE_AGENT_API_URL=$NEXT_PUBLIC_TRIPLE_AGENT_API_URL

RUN bun run build

FROM oven/bun:1-alpine AS runner

WORKDIR /app
ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1
ENV PORT=3000

RUN addgroup --system --gid 1001 bunjs \
    && adduser --system --uid 1001 -G bunjs bunjs

COPY --from=builder --chown=bunjs:bunjs /app/.next/standalone ./
COPY --from=builder /app/public ./public
COPY --from=builder --chown=bunjs:bunjs /app/.next/static ./.next/static

USER bunjs
EXPOSE 3000

# Railway injects PORT=8080 for this service. Keep the public web service on
# the port declared above so Railway's proxy and the process agree.
CMD ["sh", "-c", "PORT=3000 bun server.js"]
