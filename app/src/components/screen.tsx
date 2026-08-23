import { type PropsWithChildren } from 'react';
import { SafeAreaView, ScrollView, StyleSheet, View } from 'react-native';

import { colors, spacing } from '@/constants/theme';

type ScreenProps = PropsWithChildren<{
  scrollable?: boolean;
  padded?: boolean;
}>;

export function Screen({ children, padded = true, scrollable = false }: ScreenProps) {
  const content = (
    <View style={[styles.inner, padded && styles.padded]}>{children}</View>
  );

  return (
    <SafeAreaView style={styles.safeArea}>
      {scrollable ? (
        <ScrollView contentContainerStyle={styles.scrollContent}>{content}</ScrollView>
      ) : (
        content
      )}
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safeArea: {
    flex: 1,
    backgroundColor: colors.sand,
  },
  inner: {
    flex: 1,
  },
  padded: {
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.md,
  },
  scrollContent: {
    flexGrow: 1,
  },
});
