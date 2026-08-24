import React from 'react';
import {
  Pressable,
  StyleSheet,
  type ViewStyle,
} from 'react-native';
import { AppText } from '@/components/ui/Text';
import { colors, radius, spacing } from '@/components/ui/theme';

interface ButtonProps {
  label: string;
  onPress: () => void;
  variant?: 'primary' | 'secondary' | 'danger';
  disabled?: boolean;
  busy?: boolean;
  accessibilityLabel?: string;
  testID?: string;
}

const variantStyles: Record<
  NonNullable<ButtonProps['variant']>,
  { container: ViewStyle; textColor: string }
> = {
  primary: {
    container: { backgroundColor: colors.accent },
    textColor: '#FFFFFF',
  },
  secondary: {
    container: { backgroundColor: colors.surface, borderWidth: 1, borderColor: colors.border },
    textColor: colors.textPrimary,
  },
  danger: {
    container: { backgroundColor: colors.danger },
    textColor: '#FFFFFF',
  },
};

export function Button({
  label,
  onPress,
  variant = 'primary',
  disabled = false,
  busy = false,
  accessibilityLabel,
  testID,
}: ButtonProps): React.JSX.Element {
  const variantStyle = variantStyles[variant];
  return (
    <Pressable
      onPress={onPress}
      disabled={disabled || busy}
      style={({ pressed }) => [
        styles.base,
        variantStyle.container,
        (disabled || busy) && styles.disabled,
        pressed && !disabled && !busy && styles.pressed,
      ]}
      accessibilityRole="button"
      accessibilityLabel={accessibilityLabel ?? label}
      accessibilityState={{ disabled: disabled || busy, busy }}
      testID={testID}
    >
      <AppText weight="semibold" style={{ color: variantStyle.textColor }}>
        {busy ? `${label}…` : label}
      </AppText>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  base: {
    borderRadius: radius.md,
    paddingHorizontal: spacing.lg,
    paddingVertical: 14,
    alignItems: 'center',
    justifyContent: 'center',
    minHeight: 48,
  },
  pressed: {
    opacity: 0.85,
  },
  disabled: {
    opacity: 0.55,
  },
});
