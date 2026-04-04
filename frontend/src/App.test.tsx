import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';

describe('Test setup smoke test', () => {
  it('renders and queries DOM correctly', () => {
    render(<div data-testid="smoke">Hello Vitest</div>);
    expect(screen.getByTestId('smoke')).toBeInTheDocument();
    expect(screen.getByTestId('smoke')).toHaveTextContent('Hello Vitest');
  });
});
