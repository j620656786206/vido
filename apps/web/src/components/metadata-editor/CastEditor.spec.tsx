/**
 * CastEditor Tests (Story 3.8 - AC1)
 */

import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { CastEditor } from './CastEditor';

describe('CastEditor', () => {
  const defaultProps = {
    cast: [],
    onAdd: vi.fn(),
    onRemove: vi.fn(),
  };

  it('renders empty state', () => {
    render(<CastEditor {...defaultProps} />);
    expect(screen.getByTestId('cast-input')).toBeTruthy();
    expect(screen.getByTestId('cast-list').children.length).toBe(0);
  });

  it('renders cast members', () => {
    render(<CastEditor {...defaultProps} cast={['演員一', '演員二', '演員三']} />);

    expect(screen.getByText('演員一')).toBeTruthy();
    expect(screen.getByText('演員二')).toBeTruthy();
    expect(screen.getByText('演員三')).toBeTruthy();
  });

  it('renders custom label', () => {
    render(<CastEditor {...defaultProps} label="卡司" />);
    expect(screen.getByText('卡司')).toBeTruthy();
  });

  it('adds cast member on Enter', async () => {
    const onAdd = vi.fn();
    render(<CastEditor {...defaultProps} onAdd={onAdd} />);

    const input = screen.getByTestId('cast-input');
    await userEvent.type(input, '新演員{enter}');

    expect(onAdd).toHaveBeenCalledWith('新演員');
  });

  it('clears input after adding', async () => {
    const onAdd = vi.fn();
    render(<CastEditor {...defaultProps} onAdd={onAdd} />);

    const input = screen.getByTestId('cast-input') as HTMLInputElement;
    await userEvent.type(input, '新演員{enter}');

    expect(input.value).toBe('');
  });

  it('does not add empty name', async () => {
    const onAdd = vi.fn();
    render(<CastEditor {...defaultProps} onAdd={onAdd} />);

    const input = screen.getByTestId('cast-input');
    await userEvent.type(input, '   {enter}');

    expect(onAdd).not.toHaveBeenCalled();
  });

  it('does not add duplicate', async () => {
    const onAdd = vi.fn();
    render(<CastEditor {...defaultProps} onAdd={onAdd} cast={['已存在']} />);

    const input = screen.getByTestId('cast-input');
    await userEvent.type(input, '已存在{enter}');

    expect(onAdd).not.toHaveBeenCalled();
  });

  it('removes cast member on X click', async () => {
    const onRemove = vi.fn();
    render(<CastEditor {...defaultProps} onRemove={onRemove} cast={['演員一']} />);

    const removeButton = screen.getByLabelText('移除 演員一');
    await userEvent.click(removeButton);

    expect(onRemove).toHaveBeenCalledWith('演員一');
  });

  it('renders custom placeholder', () => {
    render(<CastEditor {...defaultProps} placeholder="輸入名字" />);

    expect(screen.getByPlaceholderText('輸入名字')).toBeTruthy();
  });
});
