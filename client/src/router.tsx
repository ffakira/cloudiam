import { createRootRoute, createRoute, createRouter } from "@tanstack/react-router";

import { AuditRoute } from "./routes/audit";
import { ApprovalsRoute } from "./routes/approvals";
import { DeviceRoute } from "./routes/device";
import { OverviewRoute } from "./routes/overview";
import { RootLayout } from "./routes/root";

const rootRoute = createRootRoute({
  component: RootLayout,
});

const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/",
  component: OverviewRoute,
});

const approvalsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/approvals",
  component: ApprovalsRoute,
});

const deviceRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/device",
  component: DeviceRoute,
});

const auditRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/audit",
  component: AuditRoute,
});

const routeTree = rootRoute.addChildren([indexRoute, approvalsRoute, deviceRoute, auditRoute]);

export const router = createRouter({ routeTree });

declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router;
  }
}
