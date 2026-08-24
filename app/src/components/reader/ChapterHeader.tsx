import React from 'react';
import { StyleSheet, View } from 'react-native';
import { AppText } from '@/components/ui/Text';
import { colors, spacing } from '@/components/ui/theme';

interface ChapterHeaderProps {
  chapterIndex: number;
  title: string | null;
}

export function ChapterHeader({
  chapterIndex,
  title,
}: ChapterHeaderProps): React.JSX.Element {
  return (
    <View style={styles.container}>
      <AppText variant="caption" style={styles.label} testID="chapter-label">
        {`Chapter ${chapterIndex + 1}`}
      </AppText>
      {title ? (
        <AppText variant="body" weight="semibold" numberOfLines={1}>
          {title}
        </AppText>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    alignItems: 'center',
    gap: spacing.xs,
    paddingVertical: spacing.sm,
  },
  label: {
    color: colors.accent,
    textTransform: 'uppercase',
    letterSpacing: 1.2,
    fontWeight: '600',
  },
});
