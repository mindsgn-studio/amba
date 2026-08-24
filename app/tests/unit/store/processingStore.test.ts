import { useProcessingStore } from '@/store/processingStore';

describe('processingStore', () => {
  beforeEach(() => {
    useProcessingStore.getState().reset();
  });

  it('starts empty', () => {
    const state = useProcessingStore.getState();
    expect(state.jobId).toBeNull();
    expect(state.bookId).toBeNull();
    expect(state.status).toBeNull();
    expect(state.progress).toBe(0);
    expect(state.error).toBeNull();
  });

  it('starts processing', () => {
    useProcessingStore
      .getState()
      .startProcessing({ jobId: 'job123', bookId: 'abc123', status: 'queued' });
    const state = useProcessingStore.getState();
    expect(state.jobId).toBe('job123');
    expect(state.bookId).toBe('abc123');
    expect(state.status).toBe('queued');
  });

  it('updates progress with clamping', () => {
    useProcessingStore.getState().setProgress(64);
    expect(useProcessingStore.getState().progress).toBe(64);
    useProcessingStore.getState().setProgress(150);
    expect(useProcessingStore.getState().progress).toBe(100);
    useProcessingStore.getState().setProgress(-5);
    expect(useProcessingStore.getState().progress).toBe(0);
  });

  it('completes at 100%', () => {
    useProcessingStore.getState().complete();
    expect(useProcessingStore.getState().status).toBe('completed');
    expect(useProcessingStore.getState().progress).toBe(100);
  });

  it('fails with an error message', () => {
    useProcessingStore.getState().fail('TTS exploded');
    const state = useProcessingStore.getState();
    expect(state.status).toBe('failed');
    expect(state.error).toBe('TTS exploded');
  });

  it('resets after completion handling', () => {
    useProcessingStore
      .getState()
      .startProcessing({ jobId: 'j', bookId: 'b', status: 'processing' });
    useProcessingStore.getState().reset();
    expect(useProcessingStore.getState().jobId).toBeNull();
    expect(useProcessingStore.getState().status).toBeNull();
  });
});
