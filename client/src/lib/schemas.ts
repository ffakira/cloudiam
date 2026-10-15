import { z } from "zod";

export const loginSchema = z.object({
  username: z.string().min(1, "Required"),
  password: z.string().min(1, "Required"),
});

export const deviceApprovalSchema = z.object({
  user_code: z.string().trim().min(1, "Required"),
  principal: z.string().trim().min(1, "Required"),
});
