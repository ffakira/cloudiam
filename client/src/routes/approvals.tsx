import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Check, Clock3, Server, ShieldCheck, UserRound, X } from "lucide-react";
import { Button } from "react-aria-components";

import { accessApprovalMutationOptions, pendingAccessRequestsQueryOptions } from "../api/query-options";
import type { AccessRequest } from "../api/types";
import { formatDateTime, formatDuration } from "../lib/format";
import { useSessionStore } from "../lib/session";

export function ApprovalsRoute() {
  return (
    <section className="space-y-5 py-2">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <div className="text-xs font-semibold uppercase tracking-[0.14em] text-[#0d9488]">Operator review</div>
          <h1 className="mt-2 text-2xl font-semibold text-slate-950">Access approvals</h1>
          <p className="mt-2 max-w-2xl text-sm leading-6 text-slate-500">
            Review pending just-in-time access requests before CloudIAM issues short-lived AWS credentials.
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          <span className="status-pill">AWS STS</span>
          <span className="status-pill status-pill-ok">Operator gated</span>
        </div>
      </div>
      <ApprovalsRoutePanel />
    </section>
  );
}

export function ApprovalsRoutePanel() {
	const authenticated = useSessionStore((state) => state.authenticated);
	const queryClient = useQueryClient();

	const pendingQuery = useQuery(pendingAccessRequestsQueryOptions(authenticated));

	const approveMutation = useMutation(accessApprovalMutationOptions(queryClient, "approve"));

	const denyMutation = useMutation(accessApprovalMutationOptions(queryClient, "deny"));

	return (
		<PendingApprovalsPanel
			authenticated={authenticated}
      requests={pendingQuery.data ?? []}
      loading={pendingQuery.isLoading}
      error={pendingQuery.error}
      approvingID={approveMutation.variables}
      denyingID={denyMutation.variables}
      onApprove={(id) => approveMutation.mutate(id)}
      onDeny={(id) => denyMutation.mutate(id)}
    />
	);
}

function PendingApprovalsPanel({
	authenticated,
	requests,
	loading,
	error,
	approvingID,
	denyingID,
	onApprove,
	onDeny,
}: {
	authenticated: boolean;
	requests: AccessRequest[];
	loading: boolean;
	error: Error | null;
	approvingID?: number;
	denyingID?: number;
	onApprove: (id: number) => void;
	onDeny: (id: number) => void;
}) {
	return (
		<section className="min-w-0 overflow-hidden rounded-md border border-slate-200 bg-white shadow-sm">
			<div className="flex flex-wrap items-center justify-between gap-3 border-b border-slate-200 px-5 py-4">
				<div>
					<h2 className="text-base font-semibold text-slate-950">Approval queue</h2>
					<p className="mt-1 text-sm text-slate-500">Requests are reviewed against the resolved target and pinned role.</p>
				</div>
				<span className={requests.length > 0 ? "status-pill status-pill-warn" : "status-pill status-pill-ok"}>
					{requests.length} pending
				</span>
			</div>
			<div className="p-5">
			{!authenticated ? <EmptyState text="Authentication required." /> : null}
			{authenticated && loading ? <EmptyState text="Loading approvals." /> : null}
			{authenticated && error ? <ErrorText error={error} /> : null}
			{authenticated && !loading && !error && requests.length === 0 ? (
				<EmptyQueue />
			) : null}
			{requests.length > 0 ? (
				<div className="grid gap-3">
					{requests.map((request) => (
						<ApprovalRequestCard
							key={request.id}
							request={request}
							approving={approvingID === request.id}
							denying={denyingID === request.id}
							onApprove={onApprove}
							onDeny={onDeny}
						/>
					))}
				</div>
			) : null}
			</div>
		</section>
	);
}

function ApprovalRequestCard({
	request,
	approving,
	denying,
	onApprove,
	onDeny,
}: {
	request: AccessRequest;
	approving: boolean;
	denying: boolean;
	onApprove: (id: number) => void;
	onDeny: (id: number) => void;
}) {
	const busy = approving || denying;

	return (
		<article className="rounded-md border border-slate-200 bg-white p-4 shadow-sm transition hover:border-slate-300">
			<div className="grid gap-4 xl:grid-cols-[1fr_auto]">
				<div className="min-w-0">
					<div className="flex flex-wrap items-center gap-2">
						<span className="inline-flex size-9 items-center justify-center rounded-md bg-[#e6f7f5] text-[#0d3b3e]">
							<UserRound className="size-4" strokeWidth={2} />
						</span>
						<div className="min-w-0">
							<div className="flex flex-wrap items-center gap-2">
								<h3 className="truncate text-base font-semibold text-slate-950">{request.principal}</h3>
								<span className="status-pill status-pill-warn">Waiting review</span>
							</div>
							<div className="mt-1 text-sm text-slate-500">{request.target}</div>
						</div>
					</div>

					<div className="mt-4 grid gap-3 md:grid-cols-3">
						<RequestFact icon={Server} label="Scope" value={`${request.provider} / ${request.account} / ${request.role}`} />
						<RequestFact icon={Clock3} label="Duration" value={formatDuration(request.requested_duration_seconds)} />
						<RequestFact icon={ShieldCheck} label="Expires" value={formatDateTime(request.expires_at)} />
					</div>

					<div className="mt-4 rounded-md border border-slate-200 bg-slate-50 px-3 py-2">
						<div className="text-xs font-semibold uppercase tracking-widest text-slate-500">Pinned role</div>
						<div className="mt-1 truncate font-mono text-xs text-slate-700" title={request.role_arn}>
							{request.role_arn}
						</div>
						{request.policy_reason ? (
							<div className="mt-2 text-sm text-slate-600">{request.policy_reason}</div>
						) : null}
					</div>
				</div>

				<div className="flex flex-col gap-2 xl:w-36 xl:justify-center">
					<Button
						className="ui-button ui-button-primary ui-button-icon"
						isDisabled={busy}
						onPress={() => onApprove(request.id)}
					>
						<Check className="size-4" strokeWidth={2} />
						{approving ? "Approving" : "Approve"}
					</Button>
					<Button
						className="ui-button ui-button-secondary ui-button-icon"
						isDisabled={busy}
						onPress={() => onDeny(request.id)}
					>
						<X className="size-4" strokeWidth={2} />
						{denying ? "Denying" : "Deny"}
					</Button>
				</div>
			</div>
		</article>
	);
}

function RequestFact({
	icon: Icon,
	label,
	value,
}: {
	icon: typeof Server;
	label: string;
	value: string;
}) {
	return (
		<div className="min-w-0 rounded-md border border-slate-200 bg-white px-3 py-2">
			<div className="flex items-center gap-2 text-xs font-semibold uppercase tracking-widest text-slate-500">
				<Icon className="size-3.5" strokeWidth={2} />
				{label}
			</div>
			<div className="mt-1 truncate text-sm font-medium text-slate-900" title={value}>{value}</div>
		</div>
	);
}

function EmptyQueue() {
	return (
		<div className="grid min-h-56 place-items-center rounded-md border border-dashed border-slate-300 bg-slate-50 px-4 py-8 text-center">
			<div>
				<div className="mx-auto flex size-11 items-center justify-center rounded-md bg-white text-[#0d3b3e] shadow-sm">
					<ShieldCheck className="size-5" strokeWidth={2} />
				</div>
				<div className="mt-4 text-sm font-semibold text-slate-950">No requests waiting for review</div>
				<div className="mt-1 max-w-md text-sm leading-6 text-slate-500">
					New approval-required access requests will appear here with their requester, AWS target, duration, and resolved role.
				</div>
			</div>
		</div>
	);
}

function EmptyState({ text }: { text: string }) {
	return (
		<div className="flex min-h-24 items-center justify-center rounded-md border border-dashed border-slate-300 bg-slate-50 px-3 py-4 text-sm text-slate-500">
			<span>{text}</span>
		</div>
	);
}

function ErrorText({ error }: { error: Error }) {
	return (
		<div className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-900">
			{error.message}
		</div>
	);
}

function buttonClass(variant: "primary" | "secondary", extra?: string) {
	const base =
		"inline-flex h-9 w-full items-center justify-center rounded-md px-4 text-sm font-medium transition disabled:cursor-not-allowed disabled:opacity-60";
	const color =
		variant === "primary"
			? "bg-[#0d3b3e] text-white hover:bg-[#155b5f]"
			: "border border-slate-300 bg-white text-slate-950 hover:bg-slate-100";
	return [base, color, extra].filter(Boolean).join(" ");
}
