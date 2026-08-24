import React from 'react';
import { ActivityIndicator, StyleSheet, View } from 'react-native';
import { AppText } from '@/components/ui/Text';
import { colors, spacing } from '@/components/ui/theme';

interface LoadingProps {
  message?: string;
}

export function Loading({ message }: LoadingProps): React.JSX.Element {
  return (
    <View style={styles.container} testID="loading">
      <ActivityIndicator size="large" color={colors.accent} />
      {message ? (
        <AppText variant="muted" style={styles.message}>
          {message}
        </AppText>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    alignItems: 'center',
    justifyContent: 'center',
    padding: spacing.xl,
    gap: spacing.md,
  },
  message: {
    textAlign: 'center',
  },
});
