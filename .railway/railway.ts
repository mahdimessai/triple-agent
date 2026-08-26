import { defineRailway, project, service } from "railway/iac";

export default defineRailway(() => {
  const web = service("web", {
    // No GitHub remote detected. `railway up` will upload this directory.
    build: "bun run build",
    start: "bun run start",
  });

  return project("triple-agent", {
    resources: [web],
  });
});
