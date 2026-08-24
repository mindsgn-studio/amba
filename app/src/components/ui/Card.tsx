import React from 'react';
import { StyleSheet, View } from 'react-native';
import { colors, radius, spacing } from '@/components/ui/theme';

interface CardProps {
  children: React.ReactNode;
  style?: object;
}

export function Card({ children, style }: CardProps): React.JSX.Element {
  return <View style={[styles.card, style]}>{children}</View>;
}

const styles = StyleSheet.create({
  card: {
    backgroundColor: colors.surface,
    borderRadius: radius.lg,
    padding: spacing.lg,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: colors.border,
    width: '100%',
  },
});
