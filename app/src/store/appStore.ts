import { create } from 'zustand';

export type AppStatus = 'initializing' | 'ready' | 'error';

interface AppState {
  status: AppStatus;
  error: string | null;
  setStatus: (status: AppStatus) => void;
  setError: (error: string | null) => void;
  reset: () => void;
}

const initialState = {
  status: 'initializing' as AppStatus,
  error: null as string | null,
};

export const useAppStore = create<AppState>((set) => ({
  ...initialState,
  setStatus: (status) => set({ status }),
  setError: (error) => set({ error }),
  reset: () => set({ ...initialState }),
}));
