import type { MicrophonePermissionState, Settings } from '@/types/models';

export function resolveInitialRoute(
  settings: Settings,
  permission: MicrophonePermissionState,
) {
  if (!settings.completedOnboarding) {
    return '/welcome';
  }

  if (permission !== 'granted') {
    return '/permissions';
  }

  return '/home';
}
