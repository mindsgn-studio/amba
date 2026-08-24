import React from 'react';
import { StyleSheet, View } from 'react-native';
import { Button } from '@/components/ui/Button';
import { AppText } from '@/components/ui/Text';
import { colors, spacing } from '@/components/ui/theme';

interface ErrorStateProps {
  title?: string;
  message: string;
  actionLabel?: string;
  onAction?: () => void;
}

export function ErrorState({
  title = 'Something went wrong',
  message,
  actionLabel,
  onAction,
}: ErrorStateProps): React.JSX.Element {
  return (
    <View style={styles.container} testID="error-state">
      <AppText variant="title" style={styles.title}>
        {title}
      </AppText>
      <AppText variant="muted" style={styles.message}>
        {message}
      </AppText>
      {actionLabel && onAction ? (
        <Button
          label={actionLabel}
          onPress={onAction}
          accessibilityLabel={actionLabel}
          testID="error-action"
        />
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    alignItems: 'center',
    padding: spacing.xl,
    gap: spacing.md,
  },
  title: {
    color: colors.danger,
    fontSize: 22,
    textAlign: 'center',
  },
  message: {
    textAlign: 'center',
  },
});
