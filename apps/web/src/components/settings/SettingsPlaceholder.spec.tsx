import React from 'react';
import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { Database, FileText, Activity, HardDrive, ArrowUpDown, Gauge } from 'lucide-react';
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
    expect(screen.getByText('此功能將在後續版本中提供')).toBeInTheDocument();
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

  // --- "此功能將在後續版本中提供" badge styling ---

  it('renders "此功能將在後續版本中提供" badge with rounded-full styling', () => {
    render(
      React.createElement(SettingsPlaceholder, {
        icon: Database,
        title: '快取管理',
        description: '管理快取資料，釋放儲存空間',
      })
    );
    const badge = screen.getByText('此功能將在後續版本中提供');
    expect(badge.tagName).toBe('SPAN');
    expect(badge).toHaveClass('rounded-full');
  });

  // --- Different icon/title/description combinations ---

  it('renders with FileText icon and logs props', () => {
    render(
      React.createElement(SettingsPlaceholder, {
        icon: FileText,
        title: '系統日誌',
        description: '查看系統日誌，排除問題',
      })
    );
    expect(screen.getByTestId('placeholder-title')).toHaveTextContent('系統日誌');
    expect(screen.getByTestId('placeholder-description')).toHaveTextContent(
      '查看系統日誌，排除問題'
    );
    expect(screen.getByTestId('placeholder-icon')).toBeInTheDocument();
  });

  it('renders with Activity icon and status props', () => {
    render(
      React.createElement(SettingsPlaceholder, {
        icon: Activity,
        title: '服務狀態',
        description: '監控外部服務連線狀態',
      })
    );
    expect(screen.getByTestId('placeholder-title')).toHaveTextContent('服務狀態');
    expect(screen.getByTestId('placeholder-description')).toHaveTextContent('監控外部服務連線狀態');
  });

  it('renders with HardDrive icon and backup props', () => {
    render(
      React.createElement(SettingsPlaceholder, {
        icon: HardDrive,
        title: '備份與還原',
        description: '備份與還原資料庫及設定',
      })
    );
    expect(screen.getByTestId('placeholder-title')).toHaveTextContent('備份與還原');
    expect(screen.getByTestId('placeholder-description')).toHaveTextContent(
      '備份與還原資料庫及設定'
    );
  });

  it('renders with ArrowUpDown icon and export props', () => {
    render(
      React.createElement(SettingsPlaceholder, {
        icon: ArrowUpDown,
        title: '匯出/匯入',
        description: '匯出或匯入媒體庫元資料',
      })
    );
    expect(screen.getByTestId('placeholder-title')).toHaveTextContent('匯出/匯入');
    expect(screen.getByTestId('placeholder-description')).toHaveTextContent(
      '匯出或匯入媒體庫元資料'
    );
  });

  it('renders with Gauge icon and performance props', () => {
    render(
      React.createElement(SettingsPlaceholder, {
        icon: Gauge,
        title: '效能監控',
        description: '查看系統效能指標與趨勢',
      })
    );
    expect(screen.getByTestId('placeholder-title')).toHaveTextContent('效能監控');
    expect(screen.getByTestId('placeholder-description')).toHaveTextContent(
      '查看系統效能指標與趨勢'
    );
  });

  // --- "此功能將在後續版本中提供" message verification ---
  // Note: The actual implementation uses "此功能將在後續版本中提供" as the badge text
  // and each placeholder has its own specific description.
  // Verify the badge always renders regardless of props.

  it('always renders "此功能將在後續版本中提供" badge regardless of props', () => {
    render(
      React.createElement(SettingsPlaceholder, {
        icon: Gauge,
        title: '效能監控',
        description: '查看系統效能指標與趨勢',
      })
    );
    expect(screen.getByText('此功能將在後續版本中提供')).toBeInTheDocument();
  });

  // --- Structure verification ---

  it('renders title as an h2 element', () => {
    render(
      React.createElement(SettingsPlaceholder, {
        icon: Database,
        title: '快取管理',
        description: '管理快取資料，釋放儲存空間',
      })
    );
    const title = screen.getByTestId('placeholder-title');
    expect(title.tagName).toBe('H2');
  });

  it('renders description as a p element', () => {
    render(
      React.createElement(SettingsPlaceholder, {
        icon: Database,
        title: '快取管理',
        description: '管理快取資料，釋放儲存空間',
      })
    );
    const desc = screen.getByTestId('placeholder-description');
    expect(desc.tagName).toBe('P');
  });

  it('icon is wrapped in a rounded background container', () => {
    render(
      React.createElement(SettingsPlaceholder, {
        icon: Database,
        title: '快取管理',
        description: '管理快取資料，釋放儲存空間',
      })
    );
    const icon = screen.getByTestId('placeholder-icon');
    const wrapper = icon.parentElement;
    expect(wrapper).toHaveClass('rounded-full');
    expect(wrapper).toHaveClass('bg-[var(--bg-secondary)]');
  });
});
