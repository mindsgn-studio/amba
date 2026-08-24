import React from 'react';
import { StyleSheet, View } from 'react-native';
import { AppText } from '@/components/ui/Text';
import { useReaderStore } from '@/store/readerStore';
import { formatTime } from '@/utils/formatting';

/**
 * Time display (current / total). Subscribes only to the two values that
 * change with the audio clock, keeping the rest of the reader static.
 */
export function ReadingProgress(): React.JSX.Element {
  const currentTime = useReaderStore((state) => state.currentTime);
  const duration = useReaderStore((state) => state.duration);

  return (
    <View style={styles.container}>
      <AppText variant="muted" testID="reading-time">
        {`${formatTime(currentTime)} / ${formatTime(duration)}`}
      </AppText>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    alignItems: 'center',
    paddingVertical: 4,
  },
});
