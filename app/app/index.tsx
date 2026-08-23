import { useEffect } from 'react';
import { ActivityIndicator, StyleSheet, Text, View } from 'react-native';
import { router } from 'expo-router';

import { colors } from '@/constants/theme';
import { Screen } from '@/components/screen';
import { useBootstrap } from '@/features/bootstrap/use-bootstrap';

export default function IndexScreen() {
  const bootstrap = useBootstrap();

  useEffect(() => {
    if (!bootstrap.isLoading) {
      router.replace(bootstrap.route as never);
    }
  }, [bootstrap.isLoading, bootstrap.route]);

  return (
    <Screen>
      <View style={styles.container}>
        <Text style={styles.brand}>Amba</Text>
        <ActivityIndicator color={colors.clay} size="large" />
        <Text style={styles.copy}>Preparing your reading space...</Text>
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    gap: 16,
  },
  brand: {
    fontSize: 42,
    fontWeight: '900',
    color: colors.ink,
  },
  copy: {
    fontSize: 16,
    color: colors.mutedInk,
  },
});
