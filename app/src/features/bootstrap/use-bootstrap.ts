import { useEffect, useState } from 'react';

import { getRecordingPermissionsAsync } from 'expo-audio';

import { getSettings, initializeDatabase, updateSettings } from '@/db/client';
import { ensureAnonymousSession } from '@/services/auth';
import { resolveInitialRoute } from '@/features/bootstrap/resolve-initial-route';
import type { MicrophonePermissionState, Settings, UserProfile } from '@/types/models';

type BootstrapState = {
  isLoading: boolean;
  route: string;
  settings: Settings | null;
  profile: UserProfile | null;
};

function normalizePermission(
  status: string,
  canAskAgain?: boolean,
): MicrophonePermissionState {
  if (status === 'granted') {
    return 'granted';
  }

  if (status === 'denied') {
    return canAskAgain ? 'denied' : 'blocked';
  }

  return 'undetermined';
}

export function useBootstrap(): BootstrapState {
  const [state, setState] = useState<BootstrapState>({
    isLoading: true,
    route: '/welcome',
    settings: null,
    profile: null,
  });

  useEffect(() => {
    let cancelled = false;

    const load = async () => {
      await initializeDatabase();

      const [settings, profile, permissionResponse] = await Promise.all([
        getSettings(),
        ensureAnonymousSession(),
        getRecordingPermissionsAsync(),
      ]);

      const permission = normalizePermission(
        permissionResponse.status,
        permissionResponse.canAskAgain,
      );

      if (settings.microphonePermission !== permission) {
        await updateSettings({ microphonePermission: permission });
      }

      const nextSettings = {
        ...settings,
        microphonePermission: permission,
      };

      if (!cancelled) {
        setState({
          isLoading: false,
          route: resolveInitialRoute(nextSettings, permission),
          settings: nextSettings,
          profile,
        });
      }
    };

    load().catch(() => {
      if (!cancelled) {
        setState({
          isLoading: false,
          route: '/welcome',
          settings: null,
          profile: null,
        });
      }
    });

    return () => {
      cancelled = true;
    };
  }, []);

  return state;
}
