import React from 'react';
import { StyleSheet, View } from 'react-native';
import { Button } from '@/components/ui/Button';
import { Card } from '@/components/ui/Card';
import { AppText } from '@/components/ui/Text';
import { colors, spacing } from '@/components/ui/theme';

export type BookCardAction = 'continue' | 'start' | 'completed';

interface BookCardProps {
  title: string;
  author?: string | null;
  action: BookCardAction;
  subtitle?: string;
  onOpen: () => void;
  onDelete: () => void;
}

const ACTION_LABELS: Record<BookCardAction, string> = {
  continue: 'Continue Reading',
  start: 'Start Reading',
  completed: 'Completed',
};

/**
 * The single active book. Shows the primary reading action plus delete.
 */
export function BookCard({
  title,
  author,
  action,
  subtitle,
  onOpen,
  onDelete,
}: BookCardProps): React.JSX.Element {
  return (
    <Card>
      <View style={styles.container}>
        <View style={styles.header}>
          <AppText variant="title" style={styles.title} numberOfLines={2}>
            {title}
          </AppText>
          {author ? (
            <AppText variant="muted" numberOfLines={1}>
              {author}
            </AppText>
          ) : null}
          {subtitle ? (
            <AppText variant="caption" style={styles.subtitle}>
              {subtitle}
            </AppText>
          ) : null}
        </View>
        <Button
          label={ACTION_LABELS[action]}
          onPress={onOpen}
          disabled={action === 'completed'}
          accessibilityLabel={ACTION_LABELS[action]}
          testID="book-action-button"
        />
        <Button
          label="Delete Book"
          onPress={onDelete}
          variant="secondary"
          accessibilityLabel="Delete Book"
          testID="delete-book-button"
        />
      </View>
    </Card>
  );
}

const styles = StyleSheet.create({
  container: {
    gap: spacing.md,
  },
  header: {
    gap: spacing.xs,
    marginBottom: spacing.sm,
    alignItems: 'center',
  },
  title: {
    fontSize: 24,
    textAlign: 'center',
    color: colors.textPrimary,
  },
  subtitle: {
    color: colors.accent,
  },
});
