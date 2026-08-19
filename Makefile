.PHONY: dev prod dev-docker prod-docker

UNAME := $(shell uname -s 2>/dev/null)

ifeq ($(UNAME),)
ifeq ($(OS),Windows_NT)
USE_POWERSHELL := 1
endif
endif

ifeq ($(USE_POWERSHELL),1)

SHELL := pwsh.exe
.SHELLFLAGS := -NoProfile -Command

dev:
	@$$go = Start-Process go -ArgumentList 'run','./server/cmd/tripleagent' -NoNewWindow -PassThru; $$web = Start-Process npm.cmd -ArgumentList 'run','dev' -NoNewWindow -PassThru; try { while (-not $$go.HasExited -and -not $$web.HasExited) { Start-Sleep -Milliseconds 200 }; $$exitCode = 0; if ($$go.HasExited -and $$go.ExitCode -ne 0) { $$exitCode = $$go.ExitCode }; if ($$web.HasExited -and $$web.ExitCode -ne 0) { $$exitCode = $$web.ExitCode }; if ($$exitCode -ne 0) { exit $$exitCode } } finally { if ($$go -and -not $$go.HasExited) { taskkill.exe /PID $$go.Id /T /F | Out-Null }; if ($$web -and -not $$web.HasExited) { taskkill.exe /PID $$web.Id /T /F | Out-Null } }

prod:
	@$$go = Start-Process go -ArgumentList 'run','./server/cmd/tripleagent' -NoNewWindow -PassThru; try { & npm.cmd run build; if ($$LASTEXITCODE -ne 0) { exit $$LASTEXITCODE }; Wait-Process -Id $$go.Id; if ($$go.ExitCode -ne 0) { exit $$go.ExitCode } } finally { if ($$go -and -not $$go.HasExited) { taskkill.exe /PID $$go.Id /T /F | Out-Null } }

else

dev:
	@set -e; \
		go run ./server/cmd/tripleagent & go_pid=$$!; \
		npm run dev & web_pid=$$!; \
		trap 'kill $$go_pid $$web_pid 2>/dev/null || true' INT TERM EXIT; \
		wait $$go_pid $$web_pid

prod:
	@set -e; \
		go run ./server/cmd/tripleagent & go_pid=$$!; \
		trap 'kill $$go_pid 2>/dev/null || true' INT TERM EXIT; \
		npm run build; \
		wait $$go_pid

endif

dev-docker:
	@docker compose -f docker-compose.yml -f docker-compose.dev.yml up --build

prod-docker:
	@docker compose up --build
