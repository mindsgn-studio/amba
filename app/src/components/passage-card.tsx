import { Pressable, StyleSheet, Text, View } from 'react-native';

import { colors, spacing } from '@/constants/theme';
import type { ReadingPassage } from '@/types/models';

type PassageCardProps = {
  passage: ReadingPassage;
  onPress: () => void;
};

export function PassageCard({ passage, onPress }: PassageCardProps) {
  return (
    <Pressable onPress={onPress} style={({ pressed }) => [styles.card, pressed && styles.pressed]}>
      <View style={styles.row}>
        <Text style={styles.category}>{passage.category}</Text>
        <Text style={styles.difficulty}>{passage.difficulty}</Text>
      </View>
      <Text style={styles.title}>{passage.title}</Text>
      <Text numberOfLines={3} style={styles.preview}>
        {passage.text}
      </Text>
      <Text style={styles.meta}>
        {passage.wordCount} words · about {passage.estimatedDurationSec}s
      </Text>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  card: {
    backgroundColor: colors.paper,
    borderRadius: 24,
    padding: spacing.lg,
    gap: spacing.sm,
  },
  pressed: {
    opacity: 0.88,
  },
  row: {
    flexDirection: 'row',
    justifyContent: 'space-between',
  },
  category: {
    color: colors.clay,
    fontSize: 13,
    fontWeight: '700',
    textTransform: 'uppercase',
    letterSpacing: 0.8,
  },
  difficulty: {
    color: colors.mutedInk,
    fontSize: 13,
    textTransform: 'capitalize',
  },
  title: {
    color: colors.ink,
    fontSize: 22,
    fontWeight: '800',
  },
  preview: {
    color: colors.mutedInk,
    fontSize: 16,
    lineHeight: 24,
  },
  meta: {
    color: colors.ink,
    fontSize: 14,
    fontWeight: '600',
  },
});
