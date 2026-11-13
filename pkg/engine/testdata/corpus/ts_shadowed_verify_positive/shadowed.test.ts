it('does not credit a locally shadowed verify', () => {
  const verify = (v: unknown) => v;
  verify(3);
});
