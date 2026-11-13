import React, { useContext } from 'react';
import { render } from '@testing-library/react';
import { Ctx, Provider } from './sut';

describe('probe components', () => {
  it('probe component used only as a JSX tag', () => {
    function Probe() {
      const value = useContext(Ctx);
      expect(value).toBe(1);
      return null;
    }
    render(
      <Provider value={1}>
        <Probe />
      </Provider>,
    );
  });

  it('component with children used as an opening tag', () => {
    function Shell(props: { children: React.ReactNode }) {
      expect(props.children).toBeTruthy();
      return <div>{props.children}</div>;
    }
    render(<Shell>text</Shell>);
  });
});
