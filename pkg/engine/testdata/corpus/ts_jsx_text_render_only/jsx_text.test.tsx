import { render } from '@testing-library/react';

it('renders the prompt with apostrophe without asserting', () => {
  render(<p>Don't have an account?</p>);
});
