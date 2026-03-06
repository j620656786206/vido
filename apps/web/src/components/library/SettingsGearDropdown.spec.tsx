import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import {
  SettingsGearDropdown,
  getStoredPreferences,
  savePreferences,
} from './SettingsGearDropdown';

describe('SettingsGearDropdown', () => {
  const defaultPrefs = {
    density: 'medium' as const,
    defaultSort: 'created_at',
    titleLanguage: 'zh-tw' as const,
  };
  const onChange = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
  });

  it('renders settings gear button', () => {
    render(<SettingsGearDropdown preferences={defaultPrefs} onPreferencesChange={onChange} />);
    expect(screen.getByTestId('settings-gear-button')).toBeInTheDocument();
  });

  it('opens dropdown on click', () => {
    render(<SettingsGearDropdown preferences={defaultPrefs} onPreferencesChange={onChange} />);
    expect(screen.queryByTestId('settings-dropdown')).not.toBeInTheDocument();

    fireEvent.click(screen.getByTestId('settings-gear-button'));
    expect(screen.getByTestId('settings-dropdown')).toBeInTheDocument();
  });

  it('shows density options', () => {
    render(<SettingsGearDropdown preferences={defaultPrefs} onPreferencesChange={onChange} />);
    fireEvent.click(screen.getByTestId('settings-gear-button'));

    expect(screen.getByText('小')).toBeInTheDocument();
    expect(screen.getByText('中')).toBeInTheDocument();
    expect(screen.getByText('大')).toBeInTheDocument();
  });

  it('changes density on click', () => {
    render(<SettingsGearDropdown preferences={defaultPrefs} onPreferencesChange={onChange} />);
    fireEvent.click(screen.getByTestId('settings-gear-button'));
    fireEvent.click(screen.getByText('大'));

    expect(onChange).toHaveBeenCalledWith({ ...defaultPrefs, density: 'large' });
  });

  it('shows title language options', () => {
    render(<SettingsGearDropdown preferences={defaultPrefs} onPreferencesChange={onChange} />);
    fireEvent.click(screen.getByTestId('settings-gear-button'));

    expect(screen.getByText('中文優先')).toBeInTheDocument();
    expect(screen.getByText('原始語言')).toBeInTheDocument();
  });

  it('persists preferences to localStorage', () => {
    savePreferences(defaultPrefs);
    const stored = getStoredPreferences();
    expect(stored).toEqual(defaultPrefs);
  });

  it('returns defaults when localStorage is empty', () => {
    const prefs = getStoredPreferences();
    expect(prefs.density).toBe('medium');
    expect(prefs.defaultSort).toBe('created_at');
    expect(prefs.titleLanguage).toBe('zh-tw');
  });
});
