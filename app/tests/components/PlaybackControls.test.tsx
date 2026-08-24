import React from 'react';
import { fireEvent, render } from '@testing-library/react-native';
import { PlaybackControls } from '@/components/reader/PlaybackControls';

const baseProps = {
  isPlaying: false,
  disabled: false,
  rate: 1,
  onTogglePlayPause: jest.fn(),
  onSeekBackward: jest.fn(),
  onSeekForward: jest.fn(),
  onRateChange: jest.fn(),
};

beforeEach(() => {
  jest.clearAllMocks();
});

describe('PlaybackControls', () => {
  it('shows Play when paused and calls play', () => {
    const { getByLabelText } = render(<PlaybackControls {...baseProps} />);
    fireEvent.press(getByLabelText('Play'));
    expect(baseProps.onTogglePlayPause).toHaveBeenCalledTimes(1);
  });

  it('shows Pause when playing', () => {
    const { getByLabelText } = render(
      <PlaybackControls {...baseProps} isPlaying />,
    );
    expect(getByLabelText('Pause')).toBeTruthy();
  });

  it('calls seek backward/forward', () => {
    const { getByLabelText } = render(<PlaybackControls {...baseProps} />);
    fireEvent.press(getByLabelText('Seek backward 15 seconds'));
    expect(baseProps.onSeekBackward).toHaveBeenCalledTimes(1);
    fireEvent.press(getByLabelText('Seek forward 15 seconds'));
    expect(baseProps.onSeekForward).toHaveBeenCalledTimes(1);
  });

  it('cycles playback speed through the button label', () => {
    const { getByLabelText, rerender } = render(
      <PlaybackControls {...baseProps} rate={1} />,
    );
    fireEvent.press(getByLabelText(/Playback speed 1/));
    expect(baseProps.onRateChange).toHaveBeenCalled();

    rerender(<PlaybackControls {...baseProps} rate={1.5} />);
    expect(getByLabelText(/Playback speed 1.5/)).toBeTruthy();
  });

  it('disables controls after completion', () => {
    const onToggle = jest.fn();
    const { getByLabelText } = render(
      <PlaybackControls {...baseProps} disabled onTogglePlayPause={onToggle} />,
    );
    fireEvent.press(getByLabelText('Play'));
    expect(onToggle).not.toHaveBeenCalled();
  });
});
