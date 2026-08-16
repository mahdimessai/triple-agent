# syntax=docker/dockerfile:1

FROM node:22-alpine AS deps

WORKDIR /app
COPY package.json package-lock.json ./
COPY web/package.json ./web/package.json
RUN npm install --global npm@11.6.2 \
    && npm ci

FROM node:22-alpine AS builder

WORKDIR /app
ENV NEXT_TELEMETRY_DISABLED=1

COPY --from=deps /app/node_modules ./node_modules
COPY package.json package-lock.json ./
COPY web ./web

ARG NEXT_PUBLIC_TRIPLE_AGENT_API_URL=http://localhost:8080
ENV NEXT_PUBLIC_TRIPLE_AGENT_API_URL=$NEXT_PUBLIC_TRIPLE_AGENT_API_URL

RUN npm run build

FROM node:22-alpine AS runner

WORKDIR /app
ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1
ENV PORT=3000

RUN addgroup --system --gid 1001 nodejs \
    && adduser --system --uid 1001 nextjs

COPY --from=builder --chown=nextjs:nodejs /app/web/.next/standalone ./
COPY --from=builder /app/web/public ./web/public
COPY --from=builder --chown=nextjs:nodejs /app/web/.next/static ./web/.next/static

USER nextjs
EXPOSE 3000

CMD ["node", "web/server.js"]
