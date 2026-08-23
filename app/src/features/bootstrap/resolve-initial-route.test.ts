import { resolveInitialRoute } from '@/features/bootstrap/resolve-initial-route';

const baseSettings = {
  completedOnboarding: false,
  microphonePermission: 'undetermined' as const,
  theme: 'sunrise' as const,
};

describe('resolveInitialRoute', () => {
  it('sends first-time users to onboarding', () => {
    expect(resolveInitialRoute(baseSettings, 'undetermined')).toBe('/welcome');
  });

  it('sends onboarded users with denied permission to permissions', () => {
    expect(
      resolveInitialRoute(
        {
          ...baseSettings,
          completedOnboarding: true,
        },
        'denied',
      ),
    ).toBe('/permissions');
  });

  it('sends fully ready users to home', () => {
    expect(
      resolveInitialRoute(
        {
          ...baseSettings,
          completedOnboarding: true,
          microphonePermission: 'granted',
        },
        'granted',
      ),
    ).toBe('/home');
  });
});
