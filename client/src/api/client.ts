import type {
  AccessRequest,
  ConsoleSession,
  AuditEvent,
  DeviceApprovalValues,
  LoginValues,
} from "./types";
import { useSessionStore } from "../lib/session";

export async function loginConsole(values: LoginValues): Promise<ConsoleSession> {
	return requestJSON<ConsoleSession>("/api/admin/login", {
		method: "POST",
		body: JSON.stringify(values),
	});
}

export async function getConsoleSession(): Promise<ConsoleSession> {
	return requestJSON<ConsoleSession>("/api/admin/me");
}

export async function refreshConsoleSession(): Promise<ConsoleSession> {
	return requestJSON<ConsoleSession>("/api/admin/refresh", {
		method: "POST",
	});
}

export async function logoutConsole(): Promise<{ status: string }> {
	return requestJSON<{ status: string }>("/api/admin/logout", {
		method: "POST",
	});
}

export async function approveDevice(values: DeviceApprovalValues): Promise<{ status: string }> {
	return requestJSON<{ status: string }>("/api/device/approve", {
		method: "POST",
		body: JSON.stringify(values),
	});
}

export async function listPendingAccessRequests(): Promise<AccessRequest[]> {
	const response = await requestJSON<{ access_requests: AccessRequest[] }>(
		"/api/admin/access-requests?status=pending",
	);
	return response.access_requests;
}

export async function listAuditEvents(): Promise<AuditEvent[]> {
	const response = await requestJSON<{ events: AuditEvent[] }>("/api/admin/audit/events?limit=50");
	return response.events;
}

export async function mutateAccessRequest(
	id: number,
	action: "approve" | "deny",
): Promise<AccessRequest> {
	return requestJSON<AccessRequest>(`/api/admin/access-requests/${id}/${action}`, {
	method: "POST",
	headers: {
		"Idempotency-Key": window.crypto.randomUUID(),
	},
  });
}

async function requestJSON<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers);
  if (!headers.has("Content-Type") && init.body) {
    headers.set("Content-Type", "application/json");
  }

	const response = await fetch(path, { ...init, credentials: "include", headers });
  if (!response.ok) {
    const message = await response
      .json()
      .then((body) => body.error)
      .catch(() => response.statusText);
    if (isConsoleSessionExpired(response.status, message, path)) {
      useSessionStore.getState().clearSession();
    }
    throw new Error(message || response.statusText);
  }
  return response.json() as Promise<T>;
}

function isConsoleSessionExpired(status: number, message: string, path: string) {
  if (!path.startsWith("/api/admin/")) {
    return false;
  }
  if (status === 401) {
    return true;
  }
  return message.toLowerCase().includes("console session is required");
}
