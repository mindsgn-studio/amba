import React from 'react';
import { Pressable, StyleSheet, View } from 'react-native';
import { AppText } from '@/components/ui/Text';
import { colors, spacing } from '@/components/ui/theme';

interface ReaderHeaderProps {
  title: string | null;
  onExit: () => void;
}

export function ReaderHeader({ title, onExit }: ReaderHeaderProps): React.JSX.Element {
  return (
    <View style={styles.container}>
      <Pressable
        onPress={onExit}
        accessibilityRole="button"
        accessibilityLabel="Stop reading and save position"
        hitSlop={12}
        testID="reader-exit-button"
      >
        <AppText weight="semibold" style={styles.exit}>
          Done
        </AppText>
      </Pressable>
      <AppText variant="muted" numberOfLines={1} style={styles.title}>
        {title ?? 'Reader'}
      </AppText>
      <View style={styles.spacer} />
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingBottom: spacing.sm,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: colors.border,
  },
  exit: {
    color: colors.accent,
    minWidth: 48,
  },
  title: {
    flex: 1,
    textAlign: 'center',
  },
  spacer: {
    minWidth: 48,
  },
});
