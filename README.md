# Triple Agent

## Local development

For host-side development, run the Go API plus Next.js frontend directly on the host:

```sh
make dev
```

The frontend is available at <http://localhost:3000> and the API at <http://localhost:8080>. Lobby state is process-local and disappears when the Go server stops, so no environment file or external service is required.

The command stops the host processes together when you press `Ctrl+C`. `make prod` uses the same host-side setup but runs `npm run build` for the frontend. For containerized workflows, use `make dev-docker` for hot-reloading development containers or `make prod-docker` for the production-shaped stack. `npm run server:dev` remains available for running only the Go server.

To run the complete production-shaped stack in Docker instead, use `npm run up` and `npm run down`.

The frontend reads `NEXT_PUBLIC_TRIPLE_AGENT_API_URL` and defaults to `http://localhost:8080`.

## Verification

```text
npm run typecheck
npm run lint
npm run build
npm run server:test
```
