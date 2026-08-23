import { Linking, StyleSheet, Text, View } from 'react-native';
import { router } from 'expo-router';
import {
  getRecordingPermissionsAsync,
  requestRecordingPermissionsAsync,
} from 'expo-audio';

import { PrimaryButton } from '@/components/primary-button';
import { Screen } from '@/components/screen';
import { colors, spacing } from '@/constants/theme';
import { updateSettings } from '@/db/client';

function mapPermission(status: string, canAskAgain?: boolean) {
  if (status === 'granted') {
    return 'granted';
  }

  if (status === 'denied') {
    return canAskAgain ? 'denied' : 'blocked';
  }

  return 'undetermined';
}

export default function PermissionsScreen() {
  const handleRequest = async () => {
    const result = await requestRecordingPermissionsAsync();
    const permission = mapPermission(result.status, result.canAskAgain);
    await updateSettings({ microphonePermission: permission });

    if (result.granted) {
      router.replace('/home');
    }
  };

  const handleRetry = async () => {
    const result = await getRecordingPermissionsAsync();
    const permission = mapPermission(result.status, result.canAskAgain);
    await updateSettings({ microphonePermission: permission });

    if (result.granted) {
      router.replace('/home');
      return;
    }

    if (!result.canAskAgain) {
      await Linking.openSettings();
    }
  };

  return (
    <Screen scrollable>
      <View style={styles.container}>
        <Text style={styles.eyebrow}>Microphone access</Text>
        <Text style={styles.title}>Amba needs your voice to score each reading session.</Text>
        <Text style={styles.body}>
          Audio is only captured while you are actively reading. Your MVP data stays on-device,
          and anonymous sign-in is used only to prepare your account for future sync.
        </Text>

        <View style={styles.callout}>
          <Text style={styles.calloutTitle}>Why we ask</Text>
          <Text style={styles.calloutBody}>Record your reading, compare spoken words, and save your score locally.</Text>
        </View>

        <View style={styles.actions}>
          <PrimaryButton label="Enable Microphone" onPress={handleRequest} />
          <PrimaryButton label="Retry or Open Settings" onPress={handleRetry} variant="secondary" />
        </View>
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    justifyContent: 'center',
    gap: spacing.lg,
  },
  eyebrow: {
    fontSize: 14,
    color: colors.clay,
    fontWeight: '800',
    textTransform: 'uppercase',
    letterSpacing: 1,
  },
  title: {
    fontSize: 36,
    lineHeight: 42,
    fontWeight: '900',
    color: colors.ink,
  },
  body: {
    fontSize: 17,
    lineHeight: 28,
    color: colors.mutedInk,
  },
  callout: {
    backgroundColor: colors.paper,
    borderRadius: 22,
    padding: spacing.lg,
    gap: 8,
  },
  calloutTitle: {
    fontSize: 18,
    fontWeight: '800',
    color: colors.ink,
  },
  calloutBody: {
    fontSize: 15,
    lineHeight: 24,
    color: colors.mutedInk,
  },
  actions: {
    gap: spacing.md,
  },
});
