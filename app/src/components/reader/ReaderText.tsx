import React, {
  useCallback,
  useEffect,
  useMemo,
  useRef,
} from 'react';
import { ScrollView, StyleSheet, Text, type LayoutChangeEvent } from 'react-native';
import { HighlightedWord } from '@/components/reader/HighlightedWord';
import { spacing } from '@/components/ui/theme';
import { useReaderStore } from '@/store/readerStore';

/** Distance from the top of the viewport at which the active word is kept. */
const SCROLL_ANCHOR_OFFSET = 140;

interface WordRange {
  start: number;
  end: number;
}

/**
 * Maps each timed word to its character range inside the chunk text by
 * scanning forward once. The text itself is rendered exactly as supplied
 * by the backend.
 */
function computeWordRanges(
  text: string,
  words: readonly { index: number; text: string }[],
): (WordRange | null)[] {
  const ranges: (WordRange | null)[] = [];
  let cursor = 0;

  for (const word of words) {
    if (word.text.length === 0) {
      ranges.push(null);
      continue;
    }
    const found = text.indexOf(word.text, cursor);
    if (found === -1) {
      // Should never happen with well-formed timings; degrade gracefully.
      ranges.push(null);
      continue;
    }
    ranges.push({ start: found, end: found + word.text.length });
    cursor = found + word.text.length;
  }

  return ranges;
}

/**
 * Renders the current chunk text with the spoken word highlighted.
 * Subscribes only to the word index and chunk data — never to the raw
 * audio clock — so ~50ms position updates do not re-render this tree.
 */
export function ReaderText(): React.JSX.Element {
  const chunkId = useReaderStore((state) => state.chunkId);
  const chunkText = useReaderStore((state) => state.chunkText);
  const words = useReaderStore((state) => state.words);
  const currentWordIndex = useReaderStore((state) => state.currentWordIndex);

  const scrollRef = useRef<ScrollView>(null);
  const layoutYRef = useRef(new Map<number, number>());

  const ranges = useMemo(
    () => (chunkText ? computeWordRanges(chunkText, words) : []),
    [chunkText, words],
  );

  useEffect(() => {
    layoutYRef.current.clear();
  }, [chunkId]);

  useEffect(() => {
    if (currentWordIndex < 0) {
      return;
    }
    const y = layoutYRef.current.get(currentWordIndex);
    if (y == null || !scrollRef.current || y < SCROLL_ANCHOR_OFFSET) {
      return;
    }
    scrollRef.current.scrollTo({
      y: y - SCROLL_ANCHOR_OFFSET,
      animated: true,
    });
  }, [currentWordIndex]);

  const handleWordLayout = useCallback(
    (index: number) => (event: LayoutChangeEvent): void => {
      layoutYRef.current.set(index, event.nativeEvent.layout.y);
    },
    [],
  );

  const segments = useMemo(() => {
    if (!chunkText) {
      return null;
    }
    const nodes: React.ReactNode[] = [];
    let cursor = 0;

    words.forEach((word, i) => {
      const range = ranges[i];
      if (!range) {
        return;
      }
      if (range.start > cursor) {
        nodes.push(
          <Text key={`t-${i}`}>{chunkText.slice(cursor, range.start)}</Text>,
        );
      }
      nodes.push(
        <HighlightedWord
          key={word.index}
          text={chunkText.slice(range.start, range.end)}
          active={i === currentWordIndex}
          onLayout={handleWordLayout(i)}
        />,
      );
      cursor = range.end;
    });

    if (cursor < chunkText.length) {
      nodes.push(<Text key="t-final">{chunkText.slice(cursor)}</Text>);
    }
    return nodes;
  }, [chunkText, words, ranges, currentWordIndex, handleWordLayout]);

  return (
    <ScrollView
      ref={scrollRef}
      style={styles.scroll}
      contentContainerStyle={styles.content}
      showsVerticalScrollIndicator={false}
    >
      <Text accessible accessibilityRole="text" testID="reader-text">
        {segments}
      </Text>
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  scroll: {
    flex: 1,
  },
  content: {
    paddingVertical: spacing.md,
    paddingHorizontal: spacing.xs,
    paddingBottom: spacing.xl,
  },
});
