import React from 'react';
import { Pressable, StyleSheet, View } from 'react-native';
import { AppText } from '@/components/ui/Text';
import { colors, spacing } from '@/components/ui/theme';

export const PLAYBACK_RATES = [0.75, 1, 1.25, 1.5, 2] as const;

interface PlaybackControlsProps {
  isPlaying: boolean;
  disabled: boolean;
  rate: number;
  onTogglePlayPause: () => void;
  onSeekBackward: () => void;
  onSeekForward: () => void;
  onRateChange: () => void;
}

function ControlButton({
  label,
  accessibilityLabel,
  onPress,
  disabled,
  large = false,
}: {
  label: string;
  accessibilityLabel: string;
  onPress: () => void;
  disabled?: boolean;
  large?: boolean;
}): React.JSX.Element {
  return (
    <Pressable
      onPress={onPress}
      disabled={disabled}
      accessibilityRole="button"
      accessibilityLabel={accessibilityLabel}
      accessibilityState={{ disabled }}
      hitSlop={8}
      style={({ pressed }) => [
        styles.control,
        large && styles.controlLarge,
        pressed && styles.pressed,
        disabled && styles.disabled,
      ]}
    >
      <AppText
        weight="semibold"
        style={large ? [styles.controlText, styles.controlTextLarge] : styles.controlText}
      >
        {label}
      </AppText>
    </Pressable>
  );
}

/**
 * Play / pause, seek and playback speed controls for the reader.
 */
export function PlaybackControls({
  isPlaying,
  disabled,
  rate,
  onTogglePlayPause,
  onSeekBackward,
  onSeekForward,
  onRateChange,
}: PlaybackControlsProps): React.JSX.Element {
  return (
    <View style={styles.container}>
      <ControlButton
        label="-15s"
        accessibilityLabel="Seek backward 15 seconds"
        onPress={onSeekBackward}
        disabled={disabled}
      />
      <ControlButton
        label={isPlaying ? 'Pause' : 'Play'}
        accessibilityLabel={isPlaying ? 'Pause' : 'Play'}
        onPress={onTogglePlayPause}
        disabled={disabled}
        large
      />
      <ControlButton
        label="+15s"
        accessibilityLabel="Seek forward 15 seconds"
        onPress={onSeekForward}
        disabled={disabled}
      />
      <ControlButton
        label={`${rate}x`}
        accessibilityLabel={`Playback speed ${rate}. Activate to change`}
        onPress={onRateChange}
        disabled={disabled}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: spacing.md,
    paddingTop: spacing.md,
  },
  control: {
    backgroundColor: colors.surface,
    borderWidth: 1,
    borderColor: colors.border,
    borderRadius: 40,
    minWidth: 64,
    minHeight: 48,
    alignItems: 'center',
    justifyContent: 'center',
    paddingHorizontal: spacing.sm,
  },
  controlLarge: {
    minWidth: 96,
    minHeight: 72,
    borderRadius: 60,
    backgroundColor: colors.accent,
    borderColor: colors.accent,
  },
  controlText: {
    color: colors.textPrimary,
  },
  controlTextLarge: {
    color: '#FFFFFF',
    fontSize: 22,
  },
  pressed: {
    opacity: 0.85,
  },
  disabled: {
    opacity: 0.4,
  },
});
