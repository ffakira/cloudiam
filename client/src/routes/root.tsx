import { Link, Outlet, useRouterState } from "@tanstack/react-router";
import { useMutation, useQuery } from "@tanstack/react-query";
import { ChevronDown, ClipboardCheck, History, LayoutDashboard, LogIn, LogOut } from "lucide-react";
import { useEffect, useState } from "react";
import type { ComponentType } from "react";

import { consoleSessionQueryOptions, logoutMutationOptions } from "../api/query-options";
import { formatDateTime } from "../lib/format";
import { useSessionStore } from "../lib/session";
import { SignInScreen } from "./sign-in";

export function RootLayout() {
  const setSession = useSessionStore((state) => state.setSession);
  const clearSession = useSessionStore((state) => state.clearSession);
  const session = useSessionStore((state) => state.session);
  const status = useSessionStore((state) => state.status);
  const pathname = useRouterState({ select: (state) => state.location.pathname });
  const sessionQuery = useQuery(consoleSessionQueryOptions());
  const logoutMutation = useMutation(logoutMutationOptions(clearSession));
  const [accountOpen, setAccountOpen] = useState(false);

  useEffect(() => {
    if (sessionQuery.data) {
      setSession(sessionQuery.data);
      return;
    }
    if (sessionQuery.isError) {
      clearSession();
    }
  }, [clearSession, sessionQuery.data, sessionQuery.isError, setSession]);

  if (pathname === "/device") {
    return <Outlet />;
  }

  if (!session && (status === "unknown" || sessionQuery.isLoading)) {
    return (
      <main className="grid min-h-screen place-items-center bg-[#eef2f1] px-4 text-slate-950">
        <div className="rounded-md border border-slate-200 bg-white px-5 py-4 text-sm font-medium text-slate-600 shadow-sm">
          Loading CloudIAM
        </div>
      </main>
    );
  }

  if (!session) {
    return <SignInScreen />;
  }

  return (
    <main className="min-h-screen bg-[#eef2f1] text-slate-950">
      <div className="grid min-h-screen w-full lg:grid-cols-[248px_minmax(0,1fr)]">
        <aside className="hidden border-r border-slate-200 bg-[#0b2224] text-white lg:sticky lg:top-0 lg:block lg:h-screen">
          <div className="flex h-full flex-col">
            <div className="border-b border-white/10 px-5 py-5">
              <Link to="/" className="flex items-center gap-3">
                <span className="flex size-10 shrink-0 items-center justify-center rounded-md bg-[#30c7b5] text-sm font-black text-[#062124]">
                  CI
                </span>
                <span>
                  <span className="block text-sm font-semibold">CloudIAM</span>
                  <span className="block text-xs text-slate-300">Access Operations</span>
                </span>
              </Link>
            </div>
            <nav className="space-y-1 px-3 py-4 text-sm">
              <NavLink to="/" icon={LayoutDashboard}>Overview</NavLink>
              <NavLink to="/approvals" icon={ClipboardCheck}>Approvals</NavLink>
              <NavLink to="/audit" icon={History}>Audit Trail</NavLink>
            </nav>
            <div className="mt-auto border-t border-white/10 p-4">
              <div className="rounded-md border border-white/10 bg-white/6 p-3">
                <div className="text-xs font-medium uppercase tracking-[0.12em] text-slate-400">Provider</div>
                <div className="mt-2 flex items-center justify-between gap-3">
                  <span className="text-sm font-semibold">AWS</span>
                  <span className="rounded-full bg-[#30c7b5]/15 px-2 py-1 text-xs font-semibold text-[#7af0df]">Active</span>
                </div>
              </div>
            </div>
          </div>
        </aside>

        <section className="min-w-0">
          <header className="sticky top-0 z-10 border-b border-slate-200 bg-white/95 px-4 py-3 backdrop-blur sm:px-6 lg:px-8">
            <div className="flex min-h-11 flex-wrap items-center justify-between gap-3">
              <Link to="/" className="flex items-center gap-3 lg:hidden">
                <span className="flex size-9 shrink-0 items-center justify-center rounded-md bg-[#0b2224] text-sm font-bold text-white">
                  CI
                </span>
                <span className="text-sm font-semibold">CloudIAM</span>
              </Link>
              <div className="hidden min-w-0 lg:block">
                <div className="text-sm font-semibold text-slate-950">Access Operations</div>
                <div className="mt-0.5 text-xs text-slate-500">AWS-only environment</div>
              </div>
              <nav className="flex flex-wrap items-center gap-1 rounded-md border border-slate-200 bg-slate-50 p-1 text-sm lg:hidden">
                <MobileNavLink to="/">Overview</MobileNavLink>
                <MobileNavLink to="/approvals">Approvals</MobileNavLink>
                <MobileNavLink to="/audit">Audit</MobileNavLink>
              </nav>
              <div className="flex items-center gap-2">
                <Link
                  to="/device"
                  className="inline-flex h-9 items-center gap-2 rounded-md border border-slate-300 bg-white px-3 text-sm font-semibold text-slate-700 transition hover:border-slate-400 hover:bg-slate-50"
                >
                  <LogIn className="size-4" strokeWidth={2} />
                  Approve device
                </Link>
                <div className="relative hidden sm:block">
                  <button
                    type="button"
                    className="flex h-9 items-center gap-2 rounded-md border border-slate-200 bg-slate-50 px-2.5 text-sm font-medium text-slate-800 transition hover:bg-white"
                    onClick={() => setAccountOpen((open) => !open)}
                  >
                    <span className="flex size-6 items-center justify-center rounded-full bg-[#0b2224] text-[11px] font-semibold uppercase text-white">
                      {initials(session.subject)}
                    </span>
                    <span className="max-w-37.5 truncate">{session.subject}</span>
                    <ChevronDown className="size-4 text-slate-500" strokeWidth={2} />
                  </button>
                  {accountOpen ? (
                    <div className="absolute right-0 top-11 w-72 overflow-hidden rounded-md border border-slate-200 bg-white shadow-lg">
                      <div className="border-b border-slate-100 px-4 py-3">
                        <div className="truncate text-sm font-semibold text-slate-950">{session.subject}</div>
                        <div className="mt-1 text-xs text-slate-500">{formatRole(session.role)}</div>
                      </div>
                      <div className="space-y-2 px-4 py-3 text-sm">
                        <div className="flex items-center justify-between gap-4">
                          <span className="text-slate-500">Status</span>
                          <span className="font-medium text-emerald-700">Active</span>
                        </div>
                        <div className="flex items-center justify-between gap-4">
                          <span className="text-slate-500">Expires</span>
                          <span className="font-medium text-slate-800">{formatDateTime(session.expires_at)}</span>
                        </div>
                      </div>
                      <button
                        type="button"
                        className="flex w-full items-center gap-2 border-t border-slate-100 px-4 py-3 text-left text-sm font-medium text-slate-700 transition hover:bg-slate-50"
                        onClick={() => {
                          setAccountOpen(false);
                          logoutMutation.mutate();
                        }}
                      >
                        <LogOut className="size-4" strokeWidth={2} />
                        Sign out
                      </button>
                    </div>
                  ) : null}
                </div>
              </div>
            </div>
          </header>
          <div className="px-4 py-5 sm:px-6 lg:px-8 xl:px-10">
            <Outlet />
          </div>
        </section>
      </div>
    </main>
  );
}

function initials(subject: string) {
	const clean = subject.trim();
	if (!clean) {
		return "?";
	}
	return clean.slice(0, 2).toUpperCase();
}

function formatRole(role: string) {
	return role
		.split("_")
		.map((part) => part.charAt(0).toUpperCase() + part.slice(1))
		.join(" ");
}

function NavLink({
  to,
  icon: Icon,
  children,
}: {
  to: string;
  icon: ComponentType<{ className?: string; strokeWidth?: number }>;
  children: React.ReactNode;
}) {
  return (
    <Link
      to={to}
      className="group flex items-center gap-3 rounded-md px-3 py-2.5 text-slate-300 transition hover:bg-white/10 hover:text-white [&.active]:bg-white [&.active]:text-[#0b2224]"
    >
      <Icon className="size-4 shrink-0 opacity-75 group-[.active]:opacity-100" strokeWidth={2} />
      {children}
    </Link>
  );
}

function MobileNavLink({ to, children }: { to: string; children: React.ReactNode }) {
  return (
    <Link
      to={to}
      className="rounded px-3 py-1.5 text-slate-600 transition hover:bg-white hover:text-slate-950 [&.active]:bg-[#0b2224] [&.active]:text-white"
    >
      {children}
    </Link>
  );
}
