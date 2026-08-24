import { formatPercent, formatTime } from '@/utils/formatting';

describe('formatTime', () => {
  it.each([
    [0, '00:00'],
    [5, '00:05'],
    [32.41, '00:32'],
    [60, '01:00'],
    [261, '04:21'],
    [3600, '60:00'],
  ])('%p seconds → %p', (input, expected) => {
    expect(formatTime(input)).toBe(expected);
  });

  it('handles invalid values safely', () => {
    expect(formatTime(-4)).toBe('00:00');
    expect(formatTime(NaN)).toBe('00:00');
  });
});

describe('formatPercent', () => {
  it('rounds and clamps to 0–100', () => {
    expect(formatPercent(64.2)).toBe('64%');
    expect(formatPercent(0)).toBe('0%');
    expect(formatPercent(100)).toBe('100%');
    expect(formatPercent(150)).toBe('100%');
    expect(formatPercent(-10)).toBe('0%');
  });
});
