import * as echarts from "echarts/core";
import { BarChart } from "echarts/charts";
import {
	GridComponent,
	LegendComponent,
	TooltipComponent,
	type GridComponentOption,
	type LegendComponentOption,
	type TooltipComponentOption,
} from "echarts/components";
import { CanvasRenderer } from "echarts/renderers";
import { useEffect, useMemo, useRef } from "react";

import type { AuditEvent } from "../api/types";

echarts.use([BarChart, GridComponent, LegendComponent, TooltipComponent, CanvasRenderer]);

type ChartOption = echarts.ComposeOption<
	GridComponentOption | LegendComponentOption | TooltipComponentOption
>;

type DecisionBucket = {
	label: string;
	startMs: number;
	allowed: number;
	denied: number;
	other: number;
};

export function AccessDecisionChart({ events }: { events: AuditEvent[] }) {
	const containerRef = useRef<HTMLDivElement | null>(null);
	const buckets = useMemo(() => buildBuckets(events), [events]);

	useEffect(() => {
		if (!containerRef.current) {
			return;
		}
		const chart = echarts.init(containerRef.current, undefined, { renderer: "canvas" });
		const option: ChartOption = {
			color: ["#10b981", "#ef4444", "#94a3b8"],
			tooltip: {
				trigger: "axis",
				axisPointer: { type: "shadow" },
			},
			legend: {
				top: 0,
				right: 0,
				itemWidth: 10,
				itemHeight: 10,
				textStyle: {
					color: "#475569",
					fontSize: 12,
					fontWeight: 600,
				},
			},
			grid: {
				left: 0,
				right: 0,
				top: 36,
				bottom: 0,
				containLabel: true,
			},
			xAxis: {
				type: "category",
				data: buckets.map((bucket) => bucket.label),
				axisLine: { lineStyle: { color: "#cbd5e1" } },
				axisTick: { show: false },
				axisLabel: { color: "#64748b", fontSize: 12 },
			},
			yAxis: {
				type: "value",
				minInterval: 1,
				axisLabel: { color: "#64748b", fontSize: 12 },
				splitLine: { lineStyle: { color: "#e2e8f0" } },
			},
			series: [
				{
					name: "Allowed",
					type: "bar",
					stack: "decisions",
					data: buckets.map((bucket) => bucket.allowed),
					barMaxWidth: 28,
				},
				{
					name: "Denied",
					type: "bar",
					stack: "decisions",
					data: buckets.map((bucket) => bucket.denied),
					barMaxWidth: 28,
				},
				{
					name: "Other",
					type: "bar",
					stack: "decisions",
					data: buckets.map((bucket) => bucket.other),
					barMaxWidth: 28,
				},
			],
		};

		chart.setOption(option);
		const resizeObserver = new ResizeObserver(() => chart.resize());
		resizeObserver.observe(containerRef.current);

		return () => {
			resizeObserver.disconnect();
			chart.dispose();
		};
	}, [buckets]);

	const total = buckets.reduce((sum, bucket) => sum + bucket.allowed + bucket.denied + bucket.other, 0);

	return (
		<section className="rounded-md border border-slate-200 bg-white shadow-sm">
			<div className="flex flex-wrap items-start justify-between gap-3 border-b border-slate-200 px-4 py-3">
				<div>
					<h2 className="text-base font-semibold text-slate-950">Recent access decisions</h2>
					<p className="mt-0.5 text-xs text-slate-500">Hourly summary from the latest audit events</p>
				</div>
				<span className="status-pill">{total} events</span>
			</div>
			{total > 0 ? (
				<div className="p-4">
					<div ref={containerRef} className="h-64 w-full" />
				</div>
			) : (
				<div className="p-4">
					<div className="flex min-h-40 items-center justify-center rounded-md border border-dashed border-slate-300 bg-slate-50 px-3 py-4 text-sm text-slate-500">
						No audit events to chart.
					</div>
				</div>
			)}
		</section>
	);
}

function buildBuckets(events: AuditEvent[]): DecisionBucket[] {
	const sorted = [...events].sort(
		(a, b) => new Date(a.occurred_at).getTime() - new Date(b.occurred_at).getTime(),
	);
	const bucketMap = new Map<number, DecisionBucket>();

	sorted.forEach((event) => {
		const startMs = hourStartMs(event.occurred_at);
		const bucket =
			bucketMap.get(startMs) ??
			{
				label: formatBucketLabel(startMs),
				startMs,
				allowed: 0,
				denied: 0,
				other: 0,
			};
		const decision = classifyDecision(event.decision);
		bucket[decision] += 1;
		bucketMap.set(startMs, bucket);
	});

	return [...bucketMap.values()].sort((a, b) => a.startMs - b.startMs).slice(-8);
}

function classifyDecision(decision: string): "allowed" | "denied" | "other" {
	const normalized = decision.toLowerCase();
	if (normalized.includes("approve") || normalized.includes("allow") || normalized.includes("issued")) {
		return "allowed";
	}
	if (normalized.includes("deny") || normalized.includes("reject") || normalized.includes("fail")) {
		return "denied";
	}
	return "other";
}

function hourStartMs(value: string) {
	const date = new Date(value);
	date.setMinutes(0, 0, 0);
	return date.getTime();
}

function formatBucketLabel(value: number) {
	return new Intl.DateTimeFormat(undefined, {
		month: "short",
		day: "numeric",
		hour: "2-digit",
	}).format(new Date(value));
}
