import React from 'react';
import {
  StyleSheet,
  Text,
  type LayoutChangeEvent,
  type StyleProp,
  type TextStyle,
} from 'react-native';
import { colors, typography } from '@/components/ui/theme';

interface HighlightedWordProps {
  text: string;
  active: boolean;
  onLayout?: (event: LayoutChangeEvent) => void;
}

const baseStyle: StyleProp<TextStyle> = {
  fontSize: typography.bodySize + 4,
  lineHeight: (typography.bodySize + 4) * 1.55,
  color: colors.textPrimary,
};

/**
 * A single word in the reading text. Memoized so only the newly
 * highlighted word re-renders when the audio position advances.
 */
function HighlightedWordBase({
  text,
  active,
  onLayout,
}: HighlightedWordProps): React.JSX.Element {
  return (
    <Text
      style={active ? [baseStyle, styles.active] : baseStyle}
      onLayout={onLayout}
      accessibilityLabel={active ? `Currently reading: ${text}` : undefined}
    >
      {text}
    </Text>
  );
}

export const HighlightedWord = React.memo(HighlightedWordBase);

const styles = StyleSheet.create({
  active: {
    color: colors.highlightText,
    backgroundColor: colors.highlight,
    borderRadius: 4,
  },
});
