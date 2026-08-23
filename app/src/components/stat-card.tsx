import { StyleSheet, Text, View } from 'react-native';

import { colors, spacing } from '@/constants/theme';

type StatCardProps = {
  label: string;
  value: string;
};

export function StatCard({ label, value }: StatCardProps) {
  return (
    <View style={styles.card}>
      <Text style={styles.value}>{value}</Text>
      <Text style={styles.label}>{label}</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  card: {
    flex: 1,
    minWidth: '47%',
    backgroundColor: colors.paper,
    borderRadius: 20,
    padding: spacing.md,
    gap: 6,
  },
  value: {
    fontSize: 26,
    fontWeight: '800',
    color: colors.ink,
  },
  label: {
    fontSize: 14,
    color: colors.mutedInk,
  },
});
