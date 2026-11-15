import { render, screen } from '@testing-library/react';

it('renders the prompt with apostrophe and asserts it', () => {
  render(<p>Don't have an account?</p>);
  expect(screen.getByText(/Don't/)).toBeInTheDocument();
});
