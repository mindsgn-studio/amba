import React from 'react';
import { StyleSheet, View } from 'react-native';
import { Button } from '@/components/ui/Button';
import { Card } from '@/components/ui/Card';
import { AppText } from '@/components/ui/Text';
import { colors, spacing } from '@/components/ui/theme';

interface UploadCardProps {
  onUpload: () => void;
  busy?: boolean;
}

/**
 * Upload entry point. Shown only when no book exists — the single-book
 * rule is additionally enforced by the service and database layers.
 */
export function UploadCard({ onUpload, busy = false }: UploadCardProps): React.JSX.Element {
  return (
    <Card>
      <View style={styles.container}>
        <AppText variant="muted" style={styles.hint}>
          Supported formats: PDF, EPUB, TXT
        </AppText>
        <Button
          label="Upload Book"
          onPress={onUpload}
          busy={busy}
          accessibilityLabel="Upload Book"
          testID="upload-book-button"
        />
      </View>
    </Card>
  );
}

const styles = StyleSheet.create({
  container: {
    alignItems: 'center',
    gap: spacing.md,
  },
  hint: {
    color: colors.textSecondary,
  },
});
