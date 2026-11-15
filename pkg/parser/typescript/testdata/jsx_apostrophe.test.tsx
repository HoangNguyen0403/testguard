import { render, screen } from '@testing-library/react';
import { Login } from './Login';

describe('Login copy', () => {
  it('renders the signup prompt without asserting', () => {
    render(<p>Don't have an account?</p>);
  });

  it('asserts the heading', () => {
    render(<Login />);
    expect(screen.getByRole('heading')).toHaveTextContent('Sign in');
  });

  it('asserts the button', () => {
    render(<Login />);
    expect(screen.getByRole('button')).toBeEnabled();
  });
});
