import React, { useEffect } from 'react';
import { Stack, useRouter } from 'expo-router';
import { StatusBar } from 'expo-status-bar';
import { StyleSheet, View } from 'react-native';
import { Loading } from '@/components/ui/Loading';
import { ErrorState } from '@/components/ui/ErrorState';
import { useAppInitialization } from '@/hooks/useAppInitialization';
import { colors } from '@/components/ui/theme';
import { useAppStore } from '@/store/appStore';
import { useProcessingStore } from '@/store/processingStore';

/**
 * Blocks the main UI until SQLite is initialized and Zustand has been
 * hydrated from persisted state. Resumes polling after a restart by
 * redirecting to the processing screen when a pending job exists.
 */
function Gate({ children }: { children: React.ReactNode }): React.JSX.Element {
  const router = useRouter();
  const appStatus = useAppStore((state) => state.status);
  const appError = useAppStore((state) => state.error);
  const jobId = useProcessingStore((state) => state.jobId);
  const jobStatus = useProcessingStore((state) => state.status);

  useEffect(() => {
    if (
      appStatus === 'ready' &&
      jobId &&
      (jobStatus === 'queued' || jobStatus === 'processing')
    ) {
      router.replace('/processing');
    }
  }, [appStatus, jobId, jobStatus, router]);

  if (appStatus === 'initializing') {
    return <Loading message="Preparing your library…" />;
  }

  if (appStatus === 'error') {
    return (
      <View style={styles.center}>
        <ErrorState
          title="Unable to start"
          message={appError ?? 'An unexpected error occurred.'}
        />
      </View>
    );
  }

  return <>{children}</>;
}

export default function RootLayout(): React.JSX.Element {
  useAppInitialization();

  return (
    <Gate>
      <StatusBar style="dark" />
      <Stack
        screenOptions={{
          headerShown: false,
          contentStyle: { backgroundColor: colors.background },
        }}
      >
        <Stack.Screen name="index" />
        <Stack.Screen name="processing" options={{ gestureEnabled: false }} />
        <Stack.Screen name="reader" />
      </Stack>
    </Gate>
  );
}

const styles = StyleSheet.create({
  center: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: colors.background,
  },
});
