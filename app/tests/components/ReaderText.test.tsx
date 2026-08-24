import React from 'react';
import { act, render } from '@testing-library/react-native';
import { ReaderText } from '@/components/reader/ReaderText';
import { useReaderStore } from '@/store/readerStore';
import type { WordTiming } from '@/schemas/timing';

const words: WordTiming[] = [
  { index: 0, text: 'The', start: 0, end: 0.2 },
  { index: 1, text: 'quick', start: 0.21, end: 0.5 },
  { index: 2, text: 'fox', start: 0.51, end: 0.8 },
];

const text = 'The quick fox';

describe('ReaderText', () => {
  beforeEach(() => {
    useReaderStore.getState().reset();
    useReaderStore.getState().setChunk({
      chunkId: 'chunk1',
      chunkIndex: 0,
      text,
      words,
    });
  });

  it('renders the timing text exactly as supplied by the backend', () => {
    const { getByTestId, queryByText } = render(<ReaderText />);
    expect(getByTestId('reader-text')).toHaveTextContent(text);
    // No highlight before playback starts.
    expect(queryByText(/Currently reading/)).toBeNull();
  });

  it('highlights the currently spoken word', () => {
    useReaderStore.setState({ currentWordIndex: 1 });
    const { getByLabelText, queryByLabelText } = render(<ReaderText />);

    expect(getByLabelText('Currently reading: quick')).toBeTruthy();
    expect(queryByLabelText('Currently reading: The')).toBeNull();
  });

  it('moves the highlight as the word index changes', () => {
    act(() => useReaderStore.setState({ currentWordIndex: 0 }));
    const screen = render(<ReaderText />);
    expect(screen.getByLabelText('Currently reading: The')).toBeTruthy();

    act(() => useReaderStore.setState({ currentWordIndex: 2 }));
    expect(screen.getByLabelText('Currently reading: fox')).toBeTruthy();
    expect(screen.queryByLabelText('Currently reading: The')).toBeNull();
  });
});
