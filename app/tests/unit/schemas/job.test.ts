import { HealthResponseSchema, JobSchema } from '@/schemas/job';
import jobCompleted from '../../fixtures/job-completed.json';
import jobFailed from '../../fixtures/job-failed.json';
import jobProcessing from '../../fixtures/job-processing.json';
import health from '../../fixtures/health.json';

describe('JobSchema', () => {
  it('accepts processing jobs', () => {
    expect(JobSchema.safeParse(jobProcessing).success).toBe(true);
  });

  it('accepts completed and failed jobs', () => {
    expect(JobSchema.safeParse(jobCompleted).success).toBe(true);
    expect(JobSchema.safeParse(jobFailed).success).toBe(true);
  });

  it('rejects invalid status', () => {
    expect(
      JobSchema.safeParse({ ...jobProcessing, status: 'paused' }).success,
    ).toBe(false);
  });

  it('rejects progress outside 0–100', () => {
    expect(JobSchema.safeParse({ ...jobProcessing, progress: 101 }).success).toBe(
      false,
    );
    expect(JobSchema.safeParse({ ...jobProcessing, progress: -1 }).success).toBe(
      false,
    );
  });

  it('rejects missing book_id', () => {
    const { book_id, ...withoutBook } = jobProcessing;
    void book_id;
    expect(JobSchema.safeParse(withoutBook).success).toBe(false);
  });
});

describe('HealthResponseSchema', () => {
  it('accepts health payloads', () => {
    expect(HealthResponseSchema.safeParse(health).success).toBe(true);
  });

  it('rejects payloads without a version', () => {
    expect(HealthResponseSchema.safeParse({ status: 'ok' }).success).toBe(false);
  });
});
