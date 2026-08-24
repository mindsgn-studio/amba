import React from 'react';
import { StyleSheet, View } from 'react-native';
import { colors, radius } from '@/components/ui/theme';

interface ProgressBarProps {
  /** 0–100 */
  progress: number;
  accessibilityLabel?: string;
}

export function ProgressBar({
  progress,
  accessibilityLabel,
}: ProgressBarProps): React.JSX.Element {
  const clamped = Math.max(0, Math.min(100, progress));
  return (
    <View
      style={styles.track}
      accessibilityRole="progressbar"
      accessibilityLabel={accessibilityLabel}
      accessibilityValue={{ min: 0, max: 100, now: Math.round(clamped) }}
      testID="progress-bar"
    >
      <View style={[styles.fill, { width: `${clamped}%` }]} />
    </View>
  );
}

const styles = StyleSheet.create({
  track: {
    height: 10,
    borderRadius: radius.sm,
    backgroundColor: colors.progressTrack,
    overflow: 'hidden',
    width: '100%',
  },
  fill: {
    height: '100%',
    borderRadius: radius.sm,
    backgroundColor: colors.progressFill,
  },
});
