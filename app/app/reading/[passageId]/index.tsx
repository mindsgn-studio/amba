import { useLocalSearchParams, router } from 'expo-router';
import { StyleSheet, Text, View } from 'react-native';

import { PrimaryButton } from '@/components/primary-button';
import { ProgressWords } from '@/components/progress-words';
import { Screen } from '@/components/screen';
import { colors, spacing } from '@/constants/theme';
import { getPassageById } from '@/constants/passages';
import { ensureAnonymousSession } from '@/services/auth';
import { useEffect, useState } from 'react';
import type { UserProfile } from '@/types/models';
import { useReadingSession } from '@/features/reading/use-reading-session';

export default function ReadingScreen() {
  const { passageId } = useLocalSearchParams<{ passageId: string }>();
  const passage = getPassageById(passageId);
  const [profile, setProfile] = useState<UserProfile | null>(null);

  useEffect(() => {
    ensureAnonymousSession().then(setProfile).catch(() => setProfile(null));
  }, []);

  if (!passage) {
    return (
      <Screen>
        <Text>Passage not found.</Text>
      </Screen>
    );
  }

  const session = useReadingSession(passage, profile);

  const handlePrimary = async () => {
    if (session.isRecording) {
      const result = await session.stop();
      if (result) {
        router.replace(`/score/${result.id}` as never);
      }
      return;
    }

    await session.start();
  };

  const words = passage.text.split(/\s+/);

  return (
    <Screen scrollable>
      <View style={styles.container}>
        <View style={styles.topBar}>
          <Text style={styles.pill}>
            {Math.round(session.recorderState.durationMillis / 1000)}s
          </Text>
          <Text style={styles.pill}>{passage.wordCount} words</Text>
          <Text style={styles.pill}>{session.state}</Text>
        </View>

        <View style={styles.hero}>
          <Text style={styles.category}>{passage.category}</Text>
          <Text style={styles.title}>{passage.title}</Text>
          <Text style={styles.instructions}>
            Read naturally, then stop the session for scoring.
          </Text>
        </View>

        <View style={styles.reader}>
          <ProgressWords words={words} activeCount={session.progressWords} />
        </View>

        {session.error ? <Text style={styles.error}>{session.error}</Text> : null}

        <View style={styles.footer}>
          <PrimaryButton
            label={session.isRecording ? 'Stop Reading' : 'Start Reading'}
            onPress={handlePrimary}
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
  topBar: {
    flexDirection: 'row',
    gap: spacing.sm,
    flexWrap: 'wrap',
  },
  pill: {
    paddingHorizontal: 14,
    paddingVertical: 8,
    borderRadius: 999,
    backgroundColor: colors.paper,
    color: colors.ink,
    fontWeight: '700',
  },
  hero: {
    gap: spacing.sm,
  },
  category: {
    fontSize: 14,
    fontWeight: '800',
    color: colors.clay,
    textTransform: 'uppercase',
    letterSpacing: 1,
  },
  title: {
    fontSize: 32,
    fontWeight: '900',
    color: colors.ink,
  },
  instructions: {
    fontSize: 16,
    lineHeight: 24,
    color: colors.mutedInk,
  },
  reader: {
    minHeight: 260,
    backgroundColor: colors.paper,
    borderRadius: 28,
    padding: spacing.lg,
    justifyContent: 'center',
  },
  error: {
    color: colors.rose,
    fontSize: 15,
    fontWeight: '600',
  },
  footer: {
    gap: spacing.md,
  },
});
