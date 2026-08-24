import React from 'react';
import { Text as RNText, type TextStyle } from 'react-native';
import { colors, typography } from '@/components/ui/theme';

interface AppTextProps {
  children: React.ReactNode;
  variant?: 'title' | 'body' | 'caption' | 'muted';
  weight?: 'regular' | 'semibold' | 'bold';
  style?: TextStyle | TextStyle[];
  numberOfLines?: number;
  selectable?: boolean;
  testID?: string;
}

const variantStyles: Record<NonNullable<AppTextProps['variant']>, TextStyle> = {
  title: {
    fontSize: typography.titleSize,
    color: colors.textPrimary,
    lineHeight: 34,
  },
  body: {
    fontSize: typography.bodySize,
    color: colors.textPrimary,
    lineHeight: 30,
  },
  caption: {
    fontSize: typography.captionSize,
    color: colors.textSecondary,
  },
  muted: {
    fontSize: typography.bodySize - 2,
    color: colors.textSecondary,
  },
};

const weightStyles: Record<NonNullable<AppTextProps['weight']>, TextStyle> = {
  regular: { fontWeight: '400' },
  semibold: { fontWeight: '600' },
  bold: { fontWeight: '700' },
};

export function AppText({
  children,
  variant = 'body',
  weight = 'regular',
  style,
  numberOfLines,
  selectable,
  testID,
}: AppTextProps): React.JSX.Element {
  return (
    <RNText
      numberOfLines={numberOfLines}
      selectable={selectable}
      testID={testID}
      style={[variantStyles[variant], weightStyles[weight], style]}
    >
      {children}
    </RNText>
  );
}
