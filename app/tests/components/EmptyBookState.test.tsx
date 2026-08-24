import React from 'react';
import { fireEvent, render } from '@testing-library/react-native';
import { EmptyBookState } from '@/components/books/EmptyBookState';

describe('EmptyBookState', () => {
  it('shows the empty-state copy and upload button', () => {
    const onUpload = jest.fn();
    const { getByText, getByLabelText } = render(
      <EmptyBookState onUpload={onUpload} />,
    );
    expect(getByText('No book yet')).toBeTruthy();
    expect(getByText('Add a book to begin')).toBeTruthy();
    expect(getByLabelText('Upload Book')).toBeTruthy();
  });

  it('triggers upload', () => {
    const onUpload = jest.fn();
    const { getByLabelText } = render(<EmptyBookState onUpload={onUpload} />);
    fireEvent.press(getByLabelText('Upload Book'));
    expect(onUpload).toHaveBeenCalledTimes(1);
  });
});
