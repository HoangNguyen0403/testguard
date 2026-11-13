it('renders profile with both behavioral assertion and snapshot', () => {
  expect(screen.getByText('Alice')).toBeVisible();
  expect(render(<Profile />)).toMatchSnapshot();
});
