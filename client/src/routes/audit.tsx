import { useQuery } from "@tanstack/react-query";
import type { CalendarDate } from "@internationalized/date";
import { Activity, CalendarDays, ChevronLeft, ChevronRight, Clock3 } from "lucide-react";
import { useState } from "react";
import {
	Button,
	Calendar,
	CalendarCell,
	CalendarGrid,
	CalendarGridBody,
	CalendarGridHeader,
	CalendarHeaderCell,
	CalendarHeading,
} from "react-aria-components";

import { auditEventsQueryOptions } from "../api/query-options";
import type { AuditEvent } from "../api/types";
import { formatDateTime } from "../lib/format";
import { useSessionStore } from "../lib/session";

export function AuditRoute() {
  return (
    <section className="py-6">
      <AuditRoutePanel />
    </section>
  );
}

export function AuditRoutePanel() {
	const authenticated = useSessionStore((state) => state.authenticated);

	const auditQuery = useQuery(auditEventsQueryOptions(authenticated));

	return (
		<AuditEventsPanel
			authenticated={authenticated}
      events={auditQuery.data ?? []}
      loading={auditQuery.isLoading}
      error={auditQuery.error}
    />
	);
}

function AuditEventsPanel({
	authenticated,
	events,
	loading,
	error,
}: {
	authenticated: boolean;
	events: AuditEvent[];
	loading: boolean;
	error: Error | null;
}) {
	return (
		<section className="min-w-0 rounded-md border border-slate-200 bg-white shadow-sm">
			<div className="flex flex-wrap items-center justify-between gap-3 border-b border-slate-200 px-4 py-3">
				<div className="flex items-center gap-3">
					<span className="flex size-8 items-center justify-center rounded-md bg-slate-100 text-slate-600">
						<Activity className="size-4" strokeWidth={2} />
					</span>
					<div>
						<h1 className="text-base font-semibold text-slate-950">Recent audit events</h1>
						<p className="mt-0.5 text-xs text-slate-500">Latest access and approval decisions</p>
					</div>
				</div>
				<div className="flex flex-wrap items-center gap-2">
					<AuditDateControl />
					<span className="status-pill">{events.length} events</span>
				</div>
			</div>
			<div className="p-4">
			{!authenticated ? <EmptyState text="Authentication required." /> : null}
			{authenticated && loading ? <EmptyState text="Loading audit events." /> : null}
			{authenticated && error ? <ErrorText error={error} /> : null}
			{authenticated && !loading && !error && events.length === 0 ? (
				<EmptyState text="No audit events." />
			) : null}
			{events.length > 0 ? (
				<div className="overflow-hidden rounded-md border border-slate-200">
					{events.map((event) => (
						<div key={event.id} className="grid gap-3 border-b border-slate-100 px-3 py-3 last:border-b-0 md:grid-cols-[150px_minmax(0,1fr)_120px]">
							<div className="flex items-start gap-2 text-sm text-slate-500">
								<Clock3 className="mt-0.5 size-4 shrink-0 text-slate-400" strokeWidth={2} />
								<time>{formatDateTime(event.occurred_at)}</time>
							</div>
							<div className="min-w-0 space-y-2">
								<div className="flex min-w-0 flex-wrap items-center gap-2">
									<span className="truncate text-sm font-semibold text-slate-950">{formatEventType(event.event_type)}</span>
									<span className={decisionClass(event.decision)}>{formatDecision(event.decision)}</span>
								</div>
								<div className="grid gap-2 text-sm text-slate-600 sm:grid-cols-2">
									<AuditField label="Principal" value={event.principal || "Unknown"} />
									<AuditField label="Target" value={event.target || "No target"} />
								</div>
								<div className="flex min-w-0 flex-wrap gap-2">
									<MetaPill value={event.provider || "provider"} />
									<MetaPill value={event.account || "account"} />
									<MetaPill value={event.role || "role"} mono />
								</div>
								{event.reason ? <div className="text-sm text-slate-500">{event.reason}</div> : null}
							</div>
							<div className="text-right text-xs font-medium uppercase tracking-widest text-slate-400">
								#{event.id}
							</div>
						</div>
					))}
				</div>
			) : null}
			</div>
		</section>
	);
}

function AuditDateControl() {
	const [open, setOpen] = useState(false);
	const [selectedDate, setSelectedDate] = useState<CalendarDate | null>(null);

	return (
		<div className="relative">
			<Button
				className="inline-flex h-8 items-center gap-2 rounded-md border border-slate-200 bg-white px-3 text-sm font-medium text-slate-700 transition hover:bg-slate-50"
				onPress={() => setOpen((next) => !next)}
			>
				<CalendarDays className="size-4 text-slate-500" strokeWidth={2} />
				{selectedDate ? selectedDate.toString() : "Viewed date"}
			</Button>
			{open ? (
				<div className="absolute right-0 top-10 z-20 w-73 rounded-md border border-slate-200 bg-white p-3 shadow-lg">
					<Calendar
						aria-label="Audit viewed date"
						value={selectedDate}
						onChange={(date) => {
							setSelectedDate(date);
							setOpen(false);
						}}
					>
						<header className="mb-3 flex items-center justify-between gap-2">
							<Button
								slot="previous"
								className="flex size-8 items-center justify-center rounded-md border border-slate-200 text-slate-600 transition hover:bg-slate-50"
							>
								<ChevronLeft className="size-4" strokeWidth={2} />
							</Button>
							<CalendarHeading className="text-sm font-semibold text-slate-950" />
							<Button
								slot="next"
								className="flex size-8 items-center justify-center rounded-md border border-slate-200 text-slate-600 transition hover:bg-slate-50"
							>
								<ChevronRight className="size-4" strokeWidth={2} />
							</Button>
						</header>
						<CalendarGrid className="w-full border-separate border-spacing-1">
							<CalendarGridHeader>
								{(day) => (
									<CalendarHeaderCell className="h-7 text-center text-xs font-semibold text-slate-400">
										{day}
									</CalendarHeaderCell>
								)}
							</CalendarGridHeader>
							<CalendarGridBody>
								{(date) => (
									<CalendarCell
										date={date}
										className={({ isDisabled, isOutsideMonth, isSelected, isToday }) =>
											[
												"size-8 cursor-default rounded-md text-center text-sm outline-none transition",
												"data-focus-visible:ring-2 data-focus-visible:ring-[#0d3b3e]/20",
												isSelected ? "bg-[#0d3b3e] font-semibold text-white" : "text-slate-700 hover:bg-slate-100",
												isToday && !isSelected ? "font-semibold text-[#0d3b3e]" : "",
												isOutsideMonth ? "text-slate-300" : "",
												isDisabled ? "opacity-40" : "",
											].join(" ")
										}
									/>
								)}
							</CalendarGridBody>
						</CalendarGrid>
						<div className="mt-3 border-t border-slate-100 pt-3 text-xs text-slate-500">
							UI-only selection for audit review. Server filtering can be added when the API supports date ranges.
						</div>
					</Calendar>
				</div>
			) : null}
		</div>
	);
}

function AuditField({ label, value }: { label: string; value: string }) {
	return (
		<div className="min-w-0">
			<div className="text-xs font-medium uppercase tracking-widest text-slate-400">{label}</div>
			<div className="mt-0.5 truncate font-medium text-slate-700">{value}</div>
		</div>
	);
}

function MetaPill({ value, mono = false }: { value: string; mono?: boolean }) {
	return (
		<span
			className={[
				"max-w-full truncate rounded-full border border-slate-200 bg-slate-50 px-2 py-1 text-xs text-slate-500",
				mono ? "font-mono" : "font-medium",
			].join(" ")}
			title={value}
		>
			{value}
		</span>
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

function decisionClass(decision: string) {
	const normalized = decision.toLowerCase();
	if (normalized.includes("approve") || normalized.includes("allow") || normalized.includes("issued")) {
		return "status-pill status-pill-ok";
	}
	if (normalized.includes("deny") || normalized.includes("reject") || normalized.includes("fail")) {
		return "status-pill status-pill-danger";
	}
	return "status-pill";
}

function formatDecision(decision: string) {
	if (!decision) {
		return "Recorded";
	}
	return decision
		.split("_")
		.map((part) => part.charAt(0).toUpperCase() + part.slice(1))
		.join(" ");
}

function formatEventType(eventType: string) {
	if (!eventType) {
		return "Audit event";
	}
	return eventType
		.split("_")
		.map((part) => part.charAt(0).toUpperCase() + part.slice(1))
		.join(" ");
}
