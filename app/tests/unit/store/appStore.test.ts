import { useAppStore } from '@/store/appStore';

describe('appStore', () => {
  beforeEach(() => {
    useAppStore.getState().reset();
  });

  it('starts in the initializing state', () => {
    const state = useAppStore.getState();
    expect(state.status).toBe('initializing');
    expect(state.error).toBeNull();
  });

  it('transitions to ready', () => {
    useAppStore.getState().setStatus('ready');
    expect(useAppStore.getState().status).toBe('ready');
  });

  it('transitions to error with a message', () => {
    useAppStore.getState().setError('Database corrupted');
    useAppStore.getState().setStatus('error');
    expect(useAppStore.getState().status).toBe('error');
    expect(useAppStore.getState().error).toBe('Database corrupted');
  });

  it('resets to the initial state', () => {
    useAppStore.getState().setStatus('ready');
    useAppStore.getState().reset();
    expect(useAppStore.getState().status).toBe('initializing');
  });
});
