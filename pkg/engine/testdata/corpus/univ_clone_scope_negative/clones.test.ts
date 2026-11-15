describe('when admin', () => {
  beforeEach(() => login('admin'));
  it('shows the panel', () => {
    render(<Panel />);
    expect(screen.getByRole('button', { name: 'Delete' })).toBeVisible();
  });
});

describe('when owner', () => {
  beforeEach(() => login('owner'));
  it('shows the panel', () => {
    render(<Panel />);
    expect(screen.getByRole('button', { name: 'Delete' })).toBeVisible();
  });
});
