import { mutationOptions, queryOptions, type QueryClient } from "@tanstack/react-query";

import {
	approveDevice,
	getConsoleSession,
	listAuditEvents,
	listPendingAccessRequests,
	loginConsole,
	logoutConsole,
	mutateAccessRequest,
	refreshConsoleSession,
} from "./client";
import type { DeviceApprovalValues, LoginValues } from "./types";

export function consoleSessionQueryOptions() {
	return queryOptions({
		queryKey: ["console-session"],
		queryFn: getConsoleSession,
		retry: false,
	});
}

export function pendingAccessRequestsQueryOptions(authenticated: boolean) {
	return queryOptions({
		queryKey: ["access-requests", "pending"],
		queryFn: listPendingAccessRequests,
		enabled: authenticated,
	});
}

export function auditEventsQueryOptions(authenticated: boolean) {
	return queryOptions({
		queryKey: ["audit-events"],
		queryFn: listAuditEvents,
		enabled: authenticated,
	});
}

export function loginMutationOptions(onSuccess: (session: Awaited<ReturnType<typeof loginConsole>>) => void) {
  return mutationOptions({
    mutationFn: (values: LoginValues) => loginConsole(values),
    onSuccess,
  });
}

export function deviceApprovalMutationOptions(onSuccess: () => void) {
  return mutationOptions({
    mutationFn: (values: DeviceApprovalValues) => approveDevice(values),
    onSuccess,
  });
}

export function refreshSessionMutationOptions(onSuccess: (session: Awaited<ReturnType<typeof refreshConsoleSession>>) => void) {
	return mutationOptions({
		mutationFn: refreshConsoleSession,
		onSuccess,
	});
}

export function logoutMutationOptions(onSettled: () => void) {
	return mutationOptions({
		mutationFn: logoutConsole,
		onSettled,
	});
}

export function accessApprovalMutationOptions(queryClient: QueryClient, action: "approve" | "deny") {
	return mutationOptions({
		mutationFn: (id: number) => mutateAccessRequest(id, action),
		onSuccess: () => {
			void queryClient.invalidateQueries({ queryKey: ["access-requests"] });
      void queryClient.invalidateQueries({ queryKey: ["audit-events"] });
    },
  });
}
