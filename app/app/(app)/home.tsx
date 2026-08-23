import { useFocusEffect, router } from 'expo-router';
import { useCallback } from 'react';
import { ActivityIndicator, StyleSheet, Text, View } from 'react-native';

import { PassageCard } from '@/components/passage-card';
import { Screen } from '@/components/screen';
import { StatCard } from '@/components/stat-card';
import { colors, spacing } from '@/constants/theme';
import { useHomeScreenData } from '@/features/home/use-home-screen-data';

export default function HomeScreen() {
  const { isLoading, passages, recentSessions, refresh, stats } = useHomeScreenData();

  useFocusEffect(
    useCallback(() => {
      refresh().catch(() => undefined);
    }, [refresh]),
  );

  return (
    <Screen scrollable>
      <View style={styles.container}>
        <View style={styles.header}>
          <Text style={styles.greeting}>Hello, Speaker</Text>
          <Text style={styles.subhead}>Pick a passage and practise your English out loud.</Text>
        </View>

        <View style={styles.stats}>
          <StatCard label="XP" value={String(stats.totalXp)} />
          <StatCard label="Accuracy" value={`${stats.averageAccuracy.toFixed(0)}%`} />
          <StatCard label="Sessions" value={String(stats.sessionsCompleted)} />
          <StatCard label="Words read" value={String(stats.totalWordsRead)} />
        </View>

        <View style={styles.challenge}>
          <Text style={styles.challengeTitle}>Daily challenge</Text>
          <Text style={styles.challengeBody}>Read 100 words today and keep your momentum alive.</Text>
        </View>

        <View style={styles.section}>
          <Text style={styles.sectionTitle}>Reading sessions</Text>
          {isLoading ? <ActivityIndicator color={colors.clay} /> : null}
          {passages.map((passage) => (
            <PassageCard
              key={passage.id}
              passage={passage}
              onPress={() => router.push(`/reading/${passage.id}` as never)}
            />
          ))}
        </View>

        <View style={styles.section}>
          <Text style={styles.sectionTitle}>Recent scores</Text>
          {recentSessions.length ? (
            recentSessions.map((session) => (
              <View key={session.id} style={styles.recentCard}>
                <Text style={styles.recentTitle}>{session.accuracy}% accuracy</Text>
                <Text style={styles.recentBody}>
                  {session.wordsCorrect} correct words · {session.xpEarned} XP
                </Text>
              </View>
            ))
          ) : (
            <View style={styles.recentCard}>
              <Text style={styles.recentTitle}>No sessions yet</Text>
              <Text style={styles.recentBody}>Your first score will appear here after a reading session.</Text>
            </View>
          )}
        </View>
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  container: {
    gap: spacing.lg,
    paddingBottom: spacing.xxl,
  },
  header: {
    gap: spacing.sm,
  },
  greeting: {
    fontSize: 34,
    fontWeight: '900',
    color: colors.ink,
  },
  subhead: {
    fontSize: 17,
    lineHeight: 26,
    color: colors.mutedInk,
  },
  stats: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: spacing.md,
  },
  challenge: {
    backgroundColor: colors.clay,
    borderRadius: 24,
    padding: spacing.lg,
    gap: 8,
  },
  challengeTitle: {
    fontSize: 18,
    fontWeight: '800',
    color: colors.paper,
  },
  challengeBody: {
    color: colors.paper,
    fontSize: 15,
    lineHeight: 24,
  },
  section: {
    gap: spacing.md,
  },
  sectionTitle: {
    fontSize: 22,
    fontWeight: '800',
    color: colors.ink,
  },
  recentCard: {
    backgroundColor: colors.paper,
    borderRadius: 20,
    padding: spacing.md,
    gap: 4,
  },
  recentTitle: {
    color: colors.ink,
    fontSize: 16,
    fontWeight: '800',
  },
  recentBody: {
    color: colors.mutedInk,
    fontSize: 14,
  },
});
