import { useState } from 'react';
import { StyleSheet, Text, View } from 'react-native';
import { router } from 'expo-router';
import Animated, { FadeIn, FadeOutDown } from 'react-native-reanimated';

import { PrimaryButton } from '@/components/primary-button';
import { Screen } from '@/components/screen';
import { colors, spacing } from '@/constants/theme';
import { updateSettings } from '@/db/client';
import { onboardingSlides } from '@/features/onboarding/slides';
import { track } from '@/services/analytics';

export default function WelcomeScreen() {
  const [slideIndex, setSlideIndex] = useState(0);
  const slide = onboardingSlides[slideIndex];
  const isLast = slideIndex === onboardingSlides.length - 1;

  const handleNext = async () => {
    if (!isLast) {
      setSlideIndex((current) => current + 1);
      return;
    }

    await updateSettings({ completedOnboarding: true });
    track('onboarding_completed');
    router.replace('/permissions');
  };

  return (
    <Screen padded={false}>
      <View style={styles.hero}>
        <Text style={styles.brand}>Amba</Text>
        <Text style={styles.tag}>Speak with confidence, one sentence at a time.</Text>
      </View>

      <View style={styles.panel}>
        <Animated.View
          entering={FadeIn.duration(350)}
          exiting={FadeOutDown.duration(250)}
          key={slide.title}
          style={styles.slide}
        >
          <Text style={styles.title}>{slide.title}</Text>
          <Text style={styles.body}>{slide.body}</Text>
        </Animated.View>

        <View style={styles.dots}>
          {onboardingSlides.map((_, index) => (
            <View
              key={index}
              style={[styles.dot, index === slideIndex ? styles.dotActive : null]}
            />
          ))}
        </View>

        <View style={styles.actions}>
          {!isLast ? (
            <PrimaryButton
              label="Skip"
              onPress={() => setSlideIndex(onboardingSlides.length - 1)}
              variant="secondary"
            />
          ) : null}
          <PrimaryButton
            label={isLast ? 'Start Learning' : 'Continue'}
            onPress={handleNext}
          />
        </View>
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  hero: {
    flex: 1,
    backgroundColor: colors.clay,
    justifyContent: 'center',
    paddingHorizontal: spacing.xl,
    paddingTop: spacing.xxl,
  },
  brand: {
    color: colors.paper,
    fontSize: 48,
    fontWeight: '900',
  },
  tag: {
    color: colors.paper,
    fontSize: 20,
    lineHeight: 30,
    marginTop: spacing.sm,
    maxWidth: 260,
  },
  panel: {
    flex: 1,
    backgroundColor: colors.sand,
    padding: spacing.xl,
    justifyContent: 'space-between',
  },
  slide: {
    gap: spacing.md,
    minHeight: 180,
  },
  title: {
    fontSize: 34,
    lineHeight: 42,
    fontWeight: '900',
    color: colors.ink,
  },
  body: {
    fontSize: 18,
    lineHeight: 28,
    color: colors.mutedInk,
  },
  dots: {
    flexDirection: 'row',
    gap: 10,
  },
  dot: {
    width: 10,
    height: 10,
    borderRadius: 999,
    backgroundColor: colors.slate,
  },
  dotActive: {
    width: 28,
    backgroundColor: colors.gold,
  },
  actions: {
    gap: spacing.md,
  },
});
