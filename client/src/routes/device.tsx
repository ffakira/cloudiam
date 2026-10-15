import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation } from "@tanstack/react-query";
import { CheckCircle2, KeyRound, ShieldCheck, Terminal } from "lucide-react";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { Button, Input, Label, TextField } from "react-aria-components";
import type { ComponentType } from "react";
import type { z } from "zod";

import { deviceApprovalMutationOptions } from "../api/query-options";
import { deviceApprovalSchema } from "../lib/schemas";

type DeviceApprovalValues = z.infer<typeof deviceApprovalSchema>;

export function DeviceRoute() {
	return (
		<main className="min-h-screen bg-[#eef2f1] px-4 py-6 text-slate-950">
			<section className="mx-auto grid min-h-[calc(100vh-3rem)] w-full max-w-5xl overflow-hidden rounded-md border border-slate-200 bg-white shadow-sm lg:grid-cols-[1fr_420px]">
				<div className="hidden bg-[#0b2224] p-8 text-white lg:flex lg:flex-col lg:justify-between">
					<div>
						<BrandMark />
						<div className="mt-12 max-w-md">
							<div className="inline-flex items-center gap-2 rounded-full border border-white/10 bg-white/6 px-3 py-1 text-xs font-semibold uppercase tracking-[0.12em] text-[#7af0df]">
								<Terminal className="size-3.5" strokeWidth={2} />
								CLI verification
							</div>
							<h1 className="mt-5 text-3xl font-semibold leading-tight">
								Approve your CloudIAM sign-in request.
							</h1>
							<p className="mt-4 text-sm leading-6 text-slate-300">
								Enter the code shown in your terminal to connect this browser approval to your command-line session.
							</p>
						</div>
					</div>
					<div className="grid gap-3 text-sm text-slate-300">
						<SecurityPoint icon={ShieldCheck} text="Only approve a request you started yourself." />
						<SecurityPoint icon={KeyRound} text="CloudIAM issues short-lived access after policy checks." />
					</div>
				</div>
				<div className="flex items-center p-5 sm:p-8">
					<DeviceApprovalPanel variant="page" />
				</div>
			</section>
		</main>
	);
}

export function DeviceApprovalPanel({ variant = "panel" }: { variant?: "panel" | "page" }) {
	const [approvalError, setApprovalError] = useState<string | null>(null);
	const [approved, setApproved] = useState(false);
	const form = useForm<DeviceApprovalValues>({
		resolver: zodResolver(deviceApprovalSchema),
		defaultValues: { user_code: "", principal: "" },
	});

	const approveMutation = useMutation(
		deviceApprovalMutationOptions(() => form.reset({ user_code: "", principal: "" })),
	);

	return (
		<section className={variant === "page" ? "w-full" : "rounded-md border border-slate-200 bg-white shadow-sm"}>
			<div className={variant === "page" ? "mb-6" : "border-b border-slate-200 px-4 py-3"}>
				{variant === "page" ? (
					<div className="mb-8 lg:hidden">
						<BrandMark dark />
					</div>
				) : null}
				<h1 className={variant === "page" ? "text-2xl font-semibold text-slate-950" : "text-base font-semibold text-slate-950"}>
					Approve this device
				</h1>
				{variant === "page" ? (
					<p className="mt-2 text-sm leading-6 text-slate-500">
						Match the code from your terminal and confirm the account requesting access.
					</p>
				) : null}
			</div>
			<form
				className={variant === "page" ? "space-y-4" : "space-y-3 p-4"}
				onSubmit={form.handleSubmit((values) => {
					setApprovalError(null);
					setApproved(false);
					approveMutation.mutate(values, {
						onError: (error) => setApprovalError(error.message),
						onSuccess: () => setApproved(true),
						onSettled: () => approveMutation.reset(),
					});
				})}
			>
				<TextField className="space-y-1" isInvalid={Boolean(form.formState.errors.user_code)}>
					<Label className="text-sm font-medium text-slate-700">Code from terminal</Label>
					<Input
						className="ui-input font-mono text-base tracking-[0.12em]"
						type="text"
						placeholder="ABCD-1234"
						{...form.register("user_code")}
					/>
					{form.formState.errors.user_code?.message ? (
						<div className="text-sm text-red-700">{form.formState.errors.user_code.message}</div>
					) : null}
				</TextField>
				<TextField className="space-y-1" isInvalid={Boolean(form.formState.errors.principal)}>
					<Label className="text-sm font-medium text-slate-700">Work email or username</Label>
					<Input
						className="ui-input"
						type="text"
						placeholder="you@example.com"
						{...form.register("principal")}
					/>
					{form.formState.errors.principal?.message ? (
						<div className="text-sm text-red-700">{form.formState.errors.principal.message}</div>
					) : null}
				</TextField>
				{approvalError ? (
					<div className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-900">
						{approvalError}
					</div>
				) : null}
				{approved ? (
					<div className="flex gap-3 rounded-md border border-emerald-200 bg-emerald-50 px-3 py-3 text-sm text-emerald-950">
						<CheckCircle2 className="mt-0.5 size-4 shrink-0" strokeWidth={2} />
						<div>
							<div className="font-semibold">Device approved</div>
							<div className="mt-1 text-emerald-800">You can return to your terminal.</div>
						</div>
					</div>
				) : null}
				{variant === "page" ? (
					<div className="rounded-md border border-slate-200 bg-slate-50 px-3 py-3 text-xs leading-5 text-slate-500">
						If you did not start this request from the CloudIAM CLI, close this page.
					</div>
				) : null}
				<Button className="ui-button ui-button-primary" type="submit" isDisabled={approveMutation.isPending}>
					{approveMutation.isPending ? "Approving" : "Approve device"}
				</Button>
			</form>
		</section>
	);
}

function BrandMark({ dark = false }: { dark?: boolean }) {
	return (
		<div className="flex items-center gap-3">
			<span
				className={
					dark
						? "flex size-10 items-center justify-center rounded-md bg-[#0b2224] text-sm font-bold text-white"
						: "flex size-10 items-center justify-center rounded-md bg-[#30c7b5] text-sm font-black text-[#062124]"
				}
			>
				CI
			</span>
			<div>
				<div className={dark ? "text-sm font-semibold text-slate-950" : "text-sm font-semibold text-white"}>
					CloudIAM
				</div>
				<div className={dark ? "text-xs text-slate-500" : "text-xs text-slate-300"}>Device approval</div>
			</div>
		</div>
	);
}

function SecurityPoint({
	icon: Icon,
	text,
}: {
	icon: ComponentType<{ className?: string; strokeWidth?: number }>;
	text: string;
}) {
	return (
		<div className="flex items-center gap-3">
			<Icon className="size-4 text-[#7af0df]" strokeWidth={2} />
			<span>{text}</span>
		</div>
	);
}
