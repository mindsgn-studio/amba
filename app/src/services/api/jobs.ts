import { JobSchema, type Job } from '@/schemas/job';
import { apiRequest } from '@/services/api/client';

/** GET /api/jobs/{job_id} */
export function getJob(jobId: string): Promise<Job> {
  return apiRequest(`/api/jobs/${encodeURIComponent(jobId)}`, JobSchema);
}
