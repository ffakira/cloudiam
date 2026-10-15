import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation } from "@tanstack/react-query";
import { ShieldCheck } from "lucide-react";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { Button, Input, Label, TextField } from "react-aria-components";
import type { z } from "zod";

import { loginMutationOptions } from "../api/query-options";
import { loginSchema } from "../lib/schemas";
import { useSessionStore } from "../lib/session";

type LoginValues = z.infer<typeof loginSchema>;

export function SignInScreen() {
	const setSession = useSessionStore((state) => state.setSession);
	const [loginError, setLoginError] = useState<string | null>(null);
	const form = useForm<LoginValues>({
		resolver: zodResolver(loginSchema),
		defaultValues: { username: "", password: "" },
	});

	const loginMutation = useMutation(
		loginMutationOptions((nextSession) => {
			setSession(nextSession);
			form.reset({ username: "", password: "" });
		}),
	);

	return (
		<main className="grid min-h-screen bg-[#eef2f1] px-4 py-8 text-slate-950">
			<section className="mx-auto grid w-full max-w-5xl overflow-hidden rounded-md border border-slate-200 bg-white shadow-sm lg:grid-cols-[1fr_380px]">
				<div className="hidden bg-[#0b2224] p-8 text-white lg:flex lg:flex-col lg:justify-between">
					<div>
						<div className="flex items-center gap-3">
							<span className="flex size-10 items-center justify-center rounded-md bg-[#30c7b5] text-sm font-black text-[#062124]">
								CI
							</span>
							<div>
								<div className="text-sm font-semibold">CloudIAM</div>
								<div className="text-xs text-slate-300">Access Operations</div>
							</div>
						</div>
						<h1 className="mt-10 max-w-md text-3xl font-semibold leading-tight">
							Control short-lived AWS access from one approval surface.
						</h1>
					</div>
					<div className="grid gap-3 text-sm text-slate-300">
						<div className="flex items-center gap-3">
							<ShieldCheck className="size-4 text-[#7af0df]" strokeWidth={2} />
							<span>Approval queue, audit trail, and STS issuance stay behind sign-in.</span>
						</div>
					</div>
				</div>
				<div className="p-6 sm:p-8">
					<div className="mb-8 lg:hidden">
						<div className="flex items-center gap-3">
							<span className="flex size-10 items-center justify-center rounded-md bg-[#0b2224] text-sm font-bold text-white">
								CI
							</span>
							<div>
								<div className="text-sm font-semibold">CloudIAM</div>
								<div className="text-xs text-slate-500">Access Operations</div>
							</div>
						</div>
					</div>
					<div className="mb-6">
						<h2 className="text-xl font-semibold text-slate-950">Sign in</h2>
						<p className="mt-2 text-sm text-slate-500">Use your console credentials to continue.</p>
					</div>
					<form
						className="space-y-3"
						onSubmit={form.handleSubmit((values) => {
							setLoginError(null);
							loginMutation.mutate(values, {
								onError: (error) => setLoginError(error.message),
								onSettled: () => loginMutation.reset(),
							});
						})}
					>
						<TextField className="space-y-1" isInvalid={Boolean(form.formState.errors.username)}>
							<Label className="text-sm font-medium text-slate-700">Username</Label>
							<Input className="ui-input" type="text" {...form.register("username")} />
							{form.formState.errors.username?.message ? (
								<div className="text-sm text-red-700">{form.formState.errors.username.message}</div>
							) : null}
						</TextField>
						<TextField className="space-y-1" isInvalid={Boolean(form.formState.errors.password)}>
							<Label className="text-sm font-medium text-slate-700">Password</Label>
							<Input className="ui-input" type="password" {...form.register("password")} />
							{form.formState.errors.password?.message ? (
								<div className="text-sm text-red-700">{form.formState.errors.password.message}</div>
							) : null}
						</TextField>
						{loginError ? (
							<div className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-900">
								{loginError}
							</div>
						) : null}
						<Button className="ui-button ui-button-primary" type="submit" isDisabled={loginMutation.isPending}>
							{loginMutation.isPending ? "Signing in" : "Sign in"}
						</Button>
					</form>
				</div>
			</section>
		</main>
	);
}
