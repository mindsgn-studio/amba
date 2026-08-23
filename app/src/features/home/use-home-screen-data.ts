import { useCallback, useEffect, useState } from 'react';

import { getHomeStats, getSettings, listRecentSessions } from '@/db/client';
import { readingPassages } from '@/constants/passages';
import type { HomeStats, ReadingSessionResult, Settings } from '@/types/models';

export function useHomeScreenData() {
  const [stats, setStats] = useState<HomeStats>({
    totalXp: 0,
    averageAccuracy: 0,
    totalWordsRead: 0,
    sessionsCompleted: 0,
  });
  const [recentSessions, setRecentSessions] = useState<ReadingSessionResult[]>([]);
  const [settings, setSettings] = useState<Settings | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  const refresh = useCallback(async () => {
    setIsLoading(true);

    const [nextStats, nextSessions, nextSettings] = await Promise.all([
      getHomeStats(),
      listRecentSessions(),
      getSettings(),
    ]);

    setStats(nextStats);
    setRecentSessions(nextSessions);
    setSettings(nextSettings);
    setIsLoading(false);
  }, []);

  useEffect(() => {
    refresh().catch(() => {
      setIsLoading(false);
    });
  }, [refresh]);

  return {
    stats,
    recentSessions,
    settings,
    passages: readingPassages,
    isLoading,
    refresh,
  };
}
