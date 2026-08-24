import React from 'react';
import { render } from '@testing-library/react-native';
import { ProcessingProgress } from '@/components/processing/ProcessingProgress';

describe('ProcessingProgress', () => {
  it('renders the title and rounded percentage', () => {
    const { getByText, getByTestId } = render(
      <ProcessingProgress title="Example Book" progress={64.4} />,
    );
    expect(getByText('Example Book')).toBeTruthy();
    expect(getByTestId('processing-percent')).toHaveTextContent('64%');
  });

  it('clamps progress display', () => {
    const { getByTestId } = render(
      <ProcessingProgress title="B" progress={150} />,
    );
    expect(getByTestId('processing-percent')).toHaveTextContent('100%');
  });

  it('shows a default generating message', () => {
    const { getByText } = render(<ProcessingProgress title="B" progress={10} />);
    expect(getByText(/Generating audiobook/)).toBeTruthy();
  });

  it('shows failure copy in error mode and hides the bar', () => {
    const onRender = render(
      <ProcessingProgress
        title="B"
        progress={42}
        message="Unable to generate this audiobook."
        isError
      />,
    );
    expect(onRender.getByText(/Unable to generate this audiobook/)).toBeTruthy();
    expect(onRender.queryByTestId('progress-bar')).toBeNull();
  });
});
