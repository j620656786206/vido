import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { DashboardLayout } from './DashboardLayout';

describe('DashboardLayout', () => {
  it('[P1] renders children', () => {
    render(
      <DashboardLayout>
        <div data-testid="child-1">Panel 1</div>
        <div data-testid="child-2">Panel 2</div>
      </DashboardLayout>
    );

    expect(screen.getByTestId('child-1')).toBeTruthy();
    expect(screen.getByTestId('child-2')).toBeTruthy();
  });

  it('[P1] has dashboard-layout test id', () => {
    render(
      <DashboardLayout>
        <div>Content</div>
      </DashboardLayout>
    );

    expect(screen.getByTestId('dashboard-layout')).toBeTruthy();
  });

  it('[P1] applies responsive grid classes', () => {
    render(
      <DashboardLayout>
        <div>Content</div>
      </DashboardLayout>
    );

    const grid = screen.getByTestId('dashboard-grid');
    expect(grid.className).toContain('grid');
    expect(grid.className).toContain('grid-cols-1');
  });

  it('[P2] accepts additional className', () => {
    render(
      <DashboardLayout className="custom-class">
        <div>Content</div>
      </DashboardLayout>
    );

    const layout = screen.getByTestId('dashboard-layout');
    expect(layout.className).toContain('custom-class');
  });
});
