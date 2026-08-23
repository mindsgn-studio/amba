import { useEffect, useState } from 'react';
import { useLocalSearchParams, router } from 'expo-router';
import { StyleSheet, Text, View } from 'react-native';

import { PrimaryButton } from '@/components/primary-button';
import { ProgressWords } from '@/components/progress-words';
import { ScoreSummary } from '@/components/score-summary';
import { Screen } from '@/components/screen';
import { colors, spacing } from '@/constants/theme';
import { readingPassages } from '@/constants/passages';
import { getReadingSessionById } from '@/db/client';
import { buildFeedback, compareWords } from '@/features/reading/scoring';
import type { ScorePayload } from '@/types/models';

export default function ScoreScreen() {
  const { sessionId } = useLocalSearchParams<{ sessionId: string }>();
  const [score, setScore] = useState<ScorePayload | null>(null);

  useEffect(() => {
    getReadingSessionById(sessionId).then((session) => {
      if (!session) {
        setScore(null);
        return;
      }

      setScore({
        ...session,
        feedback: buildFeedback(session.accuracy),
        matches: compareWords(session.expectedText, session.transcript),
      });
    });
  }, [sessionId]);

  if (!score) {
    return (
      <Screen>
        <View style={styles.loading}>
          <Text style={styles.loadingText}>Loading score...</Text>
        </View>
      </Screen>
    );
  }

  const passage = readingPassages.find((item) => item.id === score.passageId);

  return (
    <Screen scrollable>
      <View style={styles.container}>
        <ScoreSummary score={score} />

        <View style={styles.card}>
          <Text style={styles.sectionTitle}>Word-by-word review</Text>
          <ProgressWords
            words={(passage?.text ?? score.expectedText).split(/\s+/)}
            matches={score.matches}
          />
        </View>

        <View style={styles.card}>
          <Text style={styles.sectionTitle}>Transcript</Text>
          <Text style={styles.transcript}>{score.transcript}</Text>
        </View>

        <View style={styles.actions}>
          <PrimaryButton
            label="Retry"
            onPress={() => router.replace(`/reading/${score.passageId}` as never)}
          />
          <PrimaryButton label="Return Home" onPress={() => router.replace('/home')} variant="secondary" />
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
  loading: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
  },
  loadingText: {
    color: colors.mutedInk,
    fontSize: 16,
  },
  card: {
    backgroundColor: colors.paper,
    borderRadius: 24,
    padding: spacing.lg,
    gap: spacing.md,
  },
  sectionTitle: {
    fontSize: 20,
    fontWeight: '800',
    color: colors.ink,
  },
  transcript: {
    fontSize: 16,
    lineHeight: 26,
    color: colors.mutedInk,
  },
  actions: {
    gap: spacing.md,
  },
});
