export type ConsoleSession = {
  authenticated: boolean;
  subject: string;
  role: string;
  expires_at: string;
};

export type LoginValues = {
  username: string;
  password: string;
};

export type DeviceApprovalValues = {
  user_code: string;
  principal: string;
};

export type AccessRequest = {
  id: number;
  principal: string;
  target: string;
  provider: string;
  account: string;
  role: string;
  role_arn: string;
  requested_duration_seconds: number;
  policy_reason: string;
  status: string;
  expires_at: string;
  approved_by?: string;
  denied_by?: string;
  created_at: string;
  updated_at: string;
};

export type AuditEvent = {
  id: number;
  occurred_at: string;
  event_type: string;
  principal: string;
  target: string;
  decision: string;
  reason: string;
  provider: string;
  account: string;
  role: string;
  credential_expires_at?: string;
  created_at: string;
};
