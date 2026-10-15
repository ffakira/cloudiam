import { useQuery } from "@tanstack/react-query";
import { lazy, Suspense } from "react";

import {
	auditEventsQueryOptions,
	pendingAccessRequestsQueryOptions,
} from "../api/query-options";
import { useSessionStore } from "../lib/session";
import { AuditRoutePanel } from "./audit";
import { ApprovalsRoutePanel } from "./approvals";

const AccessDecisionChart = lazy(() =>
	import("../components/access-decision-chart").then((module) => ({
		default: module.AccessDecisionChart,
	})),
);

export function OverviewRoute() {
  const authenticated = useSessionStore((state) => state.authenticated);
  const pendingQuery = useQuery(pendingAccessRequestsQueryOptions(authenticated));
  const auditQuery = useQuery(auditEventsQueryOptions(authenticated));
  const pendingCount = pendingQuery.data?.length ?? 0;
  const auditCount = auditQuery.data?.length ?? 0;

  return (
    <section className="space-y-5">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold text-slate-950">Overview</h1>
          <div className="mt-2 flex flex-wrap items-center gap-2 text-sm text-slate-500">
            <span>CloudIAM</span>
            <span className="text-slate-300">/</span>
            <span>AWS Production Boundary</span>
          </div>
        </div>
        <div className="flex flex-wrap gap-2">
          <span className="status-pill status-pill-ok">Runtime ready</span>
          <span className="status-pill">AWS only</span>
        </div>
      </div>

      <div className="grid gap-3 md:grid-cols-3">
        <MetricCard label="Pending approvals" value={String(pendingCount)} tone={pendingCount > 0 ? "warn" : "ok"} />
        <MetricCard label="Audit events" value={String(auditCount)} tone="neutral" />
        <MetricCard label="Provider" value="AWS" tone="ok" />
      </div>

      <div className="space-y-4">
        <Suspense fallback={<ChartSkeleton />}>
          <AccessDecisionChart events={auditQuery.data ?? []} />
        </Suspense>
        <ApprovalsRoutePanel />
        <AuditRoutePanel />
      </div>
    </section>
	);
}

function ChartSkeleton() {
  return (
    <section className="rounded-md border border-slate-200 bg-white shadow-sm">
      <div className="border-b border-slate-200 px-4 py-3">
        <div className="h-5 w-48 rounded bg-slate-100" />
      </div>
      <div className="p-4">
        <div className="flex min-h-64 items-center justify-center rounded-md border border-dashed border-slate-300 bg-slate-50 text-sm text-slate-500">
          Loading decision chart.
        </div>
      </div>
    </section>
  );
}

function MetricCard({ label, value, tone }: { label: string; value: string; tone: "ok" | "warn" | "neutral" }) {
  const accent = tone === "ok" ? "bg-emerald-500" : tone === "warn" ? "bg-amber-500" : "bg-slate-400";

  return (
    <div className="rounded-md border border-slate-200 bg-white p-4 shadow-sm">
      <div className="flex items-center justify-between gap-3">
        <div className="text-xs font-semibold uppercase tracking-[0.12em] text-slate-500">{label}</div>
        <span className={`size-2 rounded-full ${accent}`} />
      </div>
      <div className="mt-3 text-2xl font-semibold text-slate-950">{value}</div>
    </div>
  );
}
