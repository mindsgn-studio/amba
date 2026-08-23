import { StyleSheet, Text, View } from 'react-native';

import { colors, spacing } from '@/constants/theme';
import type { WordMatch } from '@/types/models';

type ProgressWordsProps = {
  words: string[];
  activeCount?: number;
  matches?: WordMatch[];
};

export function ProgressWords({
  activeCount = 0,
  matches,
  words,
}: ProgressWordsProps) {
  return (
    <View style={styles.container}>
      {words.map((word, index) => {
        let color: string = colors.mutedInk;
        let scale = 1;

        if (matches?.[index]) {
          const status = matches[index].status;
          color =
            status === 'correct'
              ? colors.mint
              : status === 'incorrect'
                ? colors.rose
                : colors.mutedInk;
        } else if (index < activeCount) {
          color = colors.gold;
        }

        if (index === activeCount) {
          scale = 1.08;
        }

        return (
          <Text
            key={`${word}-${index}`}
            style={[styles.word, { color, transform: [{ scale }] }]}
          >
            {word}{' '}
          </Text>
        );
      })}
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: 4,
  },
  word: {
    fontSize: 28,
    lineHeight: 40,
    fontWeight: '700',
  },
});
