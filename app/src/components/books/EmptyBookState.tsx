import React from 'react';
import { StyleSheet, View } from 'react-native';
import { Button } from '@/components/ui/Button';
import { Card } from '@/components/ui/Card';
import { AppText } from '@/components/ui/Text';
import { spacing } from '@/components/ui/theme';

interface EmptyBookStateProps {
  onUpload: () => void;
  busy?: boolean;
}

export function EmptyBookState({
  onUpload,
  busy = false,
}: EmptyBookStateProps): React.JSX.Element {
  return (
    <Card>
      <View style={styles.container}>
        <AppText variant="title" style={styles.title}>
          No book yet
        </AppText>
        <AppText variant="muted">
          Add a book to begin
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
  title: {
    fontSize: 22,
  },
});
