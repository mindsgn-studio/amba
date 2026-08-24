import React from 'react';
import { fireEvent, render } from '@testing-library/react-native';
import { BookCard } from '@/components/books/BookCard';

const baseProps = {
  title: 'Example Book',
  action: 'start' as const,
  onOpen: jest.fn(),
  onDelete: jest.fn(),
};

describe('BookCard', () => {
  it('renders the book title and subtitle', () => {
    const { getByText } = render(
      <BookCard {...baseProps} subtitle="Ready to read" />,
    );
    expect(getByText('Example Book')).toBeTruthy();
    expect(getByText('Ready to read')).toBeTruthy();
  });

  it('labels the primary action for a not-started book', () => {
    const { getByLabelText } = render(<BookCard {...baseProps} action="start" />);
    expect(getByLabelText('Start Reading')).toBeTruthy();
  });

  it('opens the reader from Continue Reading', () => {
    const onOpen = jest.fn();
    const { getByLabelText } = render(
      <BookCard {...baseProps} action="continue" onOpen={onOpen} />,
    );
    fireEvent.press(getByLabelText('Continue Reading'));
    expect(onOpen).toHaveBeenCalledTimes(1);
  });

  it('calls delete', () => {
    const onDelete = jest.fn();
    const { getByLabelText } = render(
      <BookCard {...baseProps} onDelete={onDelete} />,
    );
    fireEvent.press(getByLabelText('Delete Book'));
    expect(onDelete).toHaveBeenCalledTimes(1);
  });

  it('disables the action for completed books but keeps delete available', () => {
    const onOpen = jest.fn();
    const onDelete = jest.fn();
    const { getByLabelText } = render(
      <BookCard
        {...baseProps}
        action="completed"
        subtitle="Completed"
        onOpen={onOpen}
        onDelete={onDelete}
      />,
    );
    fireEvent.press(getByLabelText('Completed'));
    expect(onOpen).not.toHaveBeenCalled();

    fireEvent.press(getByLabelText('Delete Book'));
    expect(onDelete).toHaveBeenCalledTimes(1);
  });
});
