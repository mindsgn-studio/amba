import { StyleSheet, Text, View } from 'react-native';

import { colors, spacing } from '@/constants/theme';
import type { ScorePayload } from '@/types/models';

type ScoreSummaryProps = {
  score: ScorePayload;
};

export function ScoreSummary({ score }: ScoreSummaryProps) {
  return (
    <View style={styles.card}>
      <Text style={styles.hero}>{score.accuracy}% Accuracy</Text>
      <Text style={styles.feedback}>{score.feedback}</Text>
      <View style={styles.grid}>
        <Metric label="Correct words" value={String(score.wordsCorrect)} />
        <Metric label="Missed words" value={String(score.wordsIncorrect)} />
        <Metric label="Reading speed" value={`${score.wordsPerMinute} WPM`} />
        <Metric label="XP earned" value={`+${score.xpEarned}`} />
      </View>
    </View>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <View style={styles.metric}>
      <Text style={styles.metricValue}>{value}</Text>
      <Text style={styles.metricLabel}>{label}</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  card: {
    backgroundColor: colors.paper,
    borderRadius: 24,
    padding: spacing.lg,
    gap: spacing.md,
  },
  hero: {
    fontSize: 40,
    fontWeight: '800',
    color: colors.ink,
  },
  feedback: {
    fontSize: 16,
    color: colors.mutedInk,
    lineHeight: 24,
  },
  grid: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: spacing.md,
  },
  metric: {
    width: '47%',
    backgroundColor: colors.sand,
    borderRadius: 18,
    padding: spacing.md,
    gap: 6,
  },
  metricValue: {
    fontSize: 22,
    fontWeight: '800',
    color: colors.ink,
  },
  metricLabel: {
    fontSize: 13,
    color: colors.mutedInk,
  },
});
