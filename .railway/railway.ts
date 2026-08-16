import { defineRailway, project, service } from "railway/iac";

export default defineRailway(() => {
  const web = service("web", {
    // No GitHub remote detected. `railway up` will upload this directory.
    build: "npm run build",
    start: "npm run start --workspace triple-agent-web",
  });

  return project("triple-agent", {
    resources: [web],
  });
});
