import { z } from 'zod';

export const JOB_STATUSES = ['queued', 'processing', 'completed', 'failed'] as const;

export const JobStatusSchema = z.enum(JOB_STATUSES);

export type JobStatus = z.infer<typeof JobStatusSchema>;

export function isJobStatus(value: string): value is JobStatus {
  return (JOB_STATUSES as readonly string[]).includes(value);
}

/** GET /api/jobs/{job_id} */
export const JobSchema = z.object({
  id: z.string().min(1),
  type: z.string(),
  book_id: z.string().min(1),
  status: JobStatusSchema,
  progress: z.number().min(0).max(100),
  error: z.string().nullable(),
  created_at: z.string().min(1),
  started_at: z.string().nullable().optional(),
  completed_at: z.string().nullable().optional(),
});

export type Job = z.infer<typeof JobSchema>;

/** GET /health */
export const HealthResponseSchema = z.object({
  status: z.string(),
  version: z.string(),
});

export type HealthResponse = z.infer<typeof HealthResponseSchema>;
