type EventPayload = Record<string, string | number | boolean | null | undefined>;

export function track(eventName: string, payload?: EventPayload) {
  if (__DEV__) {
    console.log('[analytics]', eventName, payload ?? {});
  }
}
