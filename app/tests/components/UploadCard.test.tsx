import React from 'react';
import { fireEvent, render } from '@testing-library/react-native';
import { UploadCard } from '@/components/books/UploadCard';

describe('UploadCard', () => {
  it('renders the supported formats hint', () => {
    const { getByText } = render(<UploadCard onUpload={() => {}} />);
    expect(getByText(/Supported formats: PDF, EPUB, TXT/)).toBeTruthy();
  });

  it('triggers the upload callback', () => {
    const onUpload = jest.fn();
    const { getByLabelText } = render(<UploadCard onUpload={onUpload} />);
    fireEvent.press(getByLabelText('Upload Book'));
    expect(onUpload).toHaveBeenCalledTimes(1);
  });

  it('does not trigger while busy', () => {
    const onUpload = jest.fn();
    const { getByLabelText } = render(<UploadCard onUpload={onUpload} busy />);
    fireEvent.press(getByLabelText('Upload Book'));
    expect(onUpload).not.toHaveBeenCalled();
  });
});
