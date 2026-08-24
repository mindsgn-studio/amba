import { useReaderStore } from '@/store/readerStore';
import type { WordTiming } from '@/schemas/timing';

const words: WordTiming[] = [
  { index: 0, text: 'The', start: 0, end: 0.2 },
  { index: 1, text: 'quick', start: 0.21, end: 0.5 },
];

describe('readerStore', () => {
  beforeEach(() => {
    useReaderStore.getState().reset();
  });

  it('starts idle and empty', () => {
    const state = useReaderStore.getState();
    expect(state.readerStatus).toBe('idle');
    expect(state.chapterId).toBeNull();
    expect(state.chunkId).toBeNull();
    expect(state.currentWordIndex).toBe(-1);
    expect(state.isPlaying).toBe(false);
    expect(state.playbackRate).toBe(1);
  });

  it('sets chapter context', () => {
    useReaderStore
      .getState()
      .setChapter({ chapterId: 'c1', chapterIndex: 2, chapterTitle: 'Chapter 3' });
    const state = useReaderStore.getState();
    expect(state.chapterId).toBe('c1');
    expect(state.chapterIndex).toBe(2);
    expect(state.chapterTitle).toBe('Chapter 3');
  });

  it('sets chunk data and resets the word cursor', () => {
    useReaderStore.getState().setCurrentWordIndex(5);
    useReaderStore.getState().setChunk({
      chunkId: 'chunk1',
      chunkIndex: 0,
      text: 'The quick',
      words,
    });
    const state = useReaderStore.getState();
    expect(state.chunkId).toBe('chunk1');
    expect(state.chunkText).toBe('The quick');
    expect(state.words).toEqual(words);
    expect(state.currentWordIndex).toBe(-1);
  });

  it('tracks the current word and time', () => {
    useReaderStore.getState().setCurrentWordIndex(1);
    useReaderStore.getState().setCurrentTime(0.35);
    expect(useReaderStore.getState().currentWordIndex).toBe(1);
    expect(useReaderStore.getState().currentTime).toBeCloseTo(0.35);
  });

  it('plays and pauses', () => {
    useReaderStore.getState().play();
    expect(useReaderStore.getState().isPlaying).toBe(true);
    expect(useReaderStore.getState().readerStatus).toBe('playing');
    useReaderStore.getState().pause();
    expect(useReaderStore.getState().isPlaying).toBe(false);
    expect(useReaderStore.getState().readerStatus).toBe('paused');
  });

  it('sets playback rate', () => {
    useReaderStore.getState().setPlaybackRate(1.5);
    expect(useReaderStore.getState().playbackRate).toBe(1.5);
  });

  it('marks completion', () => {
    useReaderStore.getState().play();
    useReaderStore.getState().setReaderStatus('completed');
    expect(useReaderStore.getState().readerStatus).toBe('completed');
  });

  it('errors set the reader status to error', () => {
    useReaderStore.getState().setError('Unable to load audio');
    expect(useReaderStore.getState().error).toBe('Unable to load audio');
    expect(useReaderStore.getState().readerStatus).toBe('error');
  });

  it('resets everything between sessions', () => {
    useReaderStore.getState().play();
    useReaderStore.getState().setPlaybackRate(2);
    useReaderStore
      .getState()
      .setChunk({ chunkId: 'x', chunkIndex: 0, text: 't', words });
    useReaderStore.getState().reset();
    const state = useReaderStore.getState();
    expect(state.isPlaying).toBe(false);
    expect(state.playbackRate).toBe(1);
    expect(state.words).toHaveLength(0);
    expect(state.currentWordIndex).toBe(-1);
  });
});
