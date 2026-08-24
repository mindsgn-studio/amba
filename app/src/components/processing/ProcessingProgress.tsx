import React from 'react';
import { StyleSheet, View } from 'react-native';
import { Card } from '@/components/ui/Card';
import { AppText } from '@/components/ui/Text';
import { ProgressBar } from '@/components/ui/ProgressBar';
import { colors, spacing } from '@/components/ui/theme';
import { formatPercent } from '@/utils/formatting';

interface ProcessingProgressProps {
  title: string;
  progress: number;
  message?: string | null;
  isError?: boolean;
}

/**
 * Audiobook generation progress card (uploading → queued → processing).
 */
export function ProcessingProgress({
  title,
  progress,
  message,
  isError = false,
}: ProcessingProgressProps): React.JSX.Element {
  return (
    <Card>
      <View style={styles.container}>
        <AppText variant="title" style={styles.title} numberOfLines={2}>
          {title}
        </AppText>
        <AppText variant="muted" style={isError ? styles.error : undefined}>
          {message ?? 'Generating audiobook…'}
        </AppText>
        {!isError ? (
          <>
            <ProgressBar
              progress={progress}
              accessibilityLabel={`Generating audiobook, ${formatPercent(progress)} complete`}
            />
            <AppText variant="caption" testID="processing-percent">
              {formatPercent(progress)}
            </AppText>
          </>
        ) : null}
      </View>
    </Card>
  );
}

const styles = StyleSheet.create({
  container: {
    alignItems: 'center',
    gap: spacing.md,
  },
  title: {
    fontSize: 24,
    textAlign: 'center',
  },
  error: {
    color: colors.danger,
    textAlign: 'center',
  },
});
