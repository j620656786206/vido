import React from 'react';
import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { Database } from 'lucide-react';
import { SettingsPlaceholder } from './SettingsPlaceholder';

describe('SettingsPlaceholder', () => {
  it('renders the title', () => {
    render(
      React.createElement(SettingsPlaceholder, {
        icon: Database,
        title: '快取管理',
        description: '管理快取資料，釋放儲存空間',
      })
    );
    expect(screen.getByTestId('placeholder-title')).toHaveTextContent('快取管理');
  });

  it('renders the description', () => {
    render(
      React.createElement(SettingsPlaceholder, {
        icon: Database,
        title: '快取管理',
        description: '管理快取資料，釋放儲存空間',
      })
    );
    expect(screen.getByTestId('placeholder-description')).toHaveTextContent(
      '管理快取資料，釋放儲存空間'
    );
  });

  it('renders the icon', () => {
    render(
      React.createElement(SettingsPlaceholder, {
        icon: Database,
        title: '快取管理',
        description: '管理快取資料，釋放儲存空間',
      })
    );
    expect(screen.getByTestId('placeholder-icon')).toBeInTheDocument();
  });

  it('renders the coming soon badge', () => {
    render(
      React.createElement(SettingsPlaceholder, {
        icon: Database,
        title: '快取管理',
        description: '管理快取資料，釋放儲存空間',
      })
    );
    expect(screen.getByText('即將推出')).toBeInTheDocument();
  });

  it('renders the placeholder container', () => {
    render(
      React.createElement(SettingsPlaceholder, {
        icon: Database,
        title: '快取管理',
        description: '管理快取資料，釋放儲存空間',
      })
    );
    expect(screen.getByTestId('settings-placeholder')).toBeInTheDocument();
  });
});
