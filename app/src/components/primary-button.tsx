import { Pressable, StyleSheet, Text } from 'react-native';

import { colors, spacing } from '@/constants/theme';

type PrimaryButtonProps = {
  label: string;
  onPress: () => void;
  disabled?: boolean;
  variant?: 'primary' | 'secondary';
};

export function PrimaryButton({
  label,
  onPress,
  disabled = false,
  variant = 'primary',
}: PrimaryButtonProps) {
  return (
    <Pressable
      accessibilityRole="button"
      onPress={onPress}
      disabled={disabled}
      style={({ pressed }) => [
        styles.button,
        variant === 'secondary' ? styles.secondary : styles.primary,
        pressed && !disabled ? styles.pressed : null,
        disabled ? styles.disabled : null,
      ]}
    >
      <Text style={[styles.label, variant === 'secondary' ? styles.secondaryLabel : null]}>
        {label}
      </Text>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  button: {
    minHeight: 56,
    borderRadius: 18,
    justifyContent: 'center',
    alignItems: 'center',
    paddingHorizontal: spacing.lg,
  },
  primary: {
    backgroundColor: colors.ink,
  },
  secondary: {
    backgroundColor: colors.paper,
    borderWidth: 1,
    borderColor: colors.slate,
  },
  pressed: {
    opacity: 0.88,
  },
  disabled: {
    opacity: 0.5,
  },
  label: {
    color: colors.paper,
    fontSize: 16,
    fontWeight: '700',
  },
  secondaryLabel: {
    color: colors.ink,
  },
});
