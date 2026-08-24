import { useBookStore } from '@/store/bookStore';

describe('bookStore', () => {
  beforeEach(() => {
    useBookStore.getState().clear();
  });

  it('starts empty', () => {
    expect(useBookStore.getState().bookId).toBeNull();
    expect(useBookStore.getState().title).toBeNull();
    expect(useBookStore.getState().status).toBeNull();
    expect(useBookStore.getState().hasProgress).toBe(false);
    expect(useBookStore.getState().completed).toBe(false);
  });

  it('sets the book', () => {
    useBookStore.getState().setBook({
      bookId: 'abc123',
      title: 'Example Book',
      author: 'Author',
      status: 'completed',
    });
    const state = useBookStore.getState();
    expect(state.bookId).toBe('abc123');
    expect(state.title).toBe('Example Book');
    expect(state.author).toBe('Author');
    expect(state.status).toBe('completed');
  });

  it('updates status', () => {
    useBookStore.getState().setBook({ bookId: 'x', title: 'X', status: 'queued' });
    useBookStore.getState().updateStatus('processing');
    expect(useBookStore.getState().status).toBe('processing');
  });

  it('tracks progress flags', () => {
    useBookStore
      .getState()
      .setBook({ bookId: 'abc123', title: 'B', status: 'completed' });
    useBookStore.getState().setHasProgress(true);
    expect(useBookStore.getState().hasProgress).toBe(true);
    useBookStore.getState().markCompleted();
    expect(useBookStore.getState().completed).toBe(true);
  });

  it('clears all fields', () => {
    useBookStore.getState().setBook({ bookId: 'x', title: 'X', status: 'failed' });
    useBookStore.getState().setHasProgress(true);
    useBookStore.getState().clear();
    expect(useBookStore.getState().bookId).toBeNull();
    expect(useBookStore.getState().hasProgress).toBe(false);
  });
});
