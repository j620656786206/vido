/**
 * GenreSelector — 修改資訊 類型 chips (poster-upload-a AC #3)
 */
import { describe, it, expect, vi } from 'vitest';
import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { GenreSelector } from './GenreSelector';

function setup(selected: string[], options = ['動作', '劇情', '動畫']) {
  const onChange = vi.fn();
  render(
    <>
      <span id="lbl">類型</span>
      <GenreSelector labelId="lbl" selected={selected} options={options} onChange={onChange} />
    </>
  );
  return onChange;
}

describe('GenreSelector', () => {
  it('names the chip group by the field label and shows each picked genre as a chip', () => {
    setup(['劇情', '豆瓣來的類型']);
    const group = screen.getByRole('group', { name: '類型' });
    expect(within(group).getByText('劇情')).toBeInTheDocument();
    expect(within(group).getByText('豆瓣來的類型')).toBeInTheDocument();
  });

  it('removes a genre with its ×', async () => {
    const onChange = setup(['劇情', '動作']);
    await userEvent.click(screen.getByRole('button', { name: '移除類型：劇情' }));
    expect(onChange).toHaveBeenCalledWith(['動作']);
  });

  it('adds from the menu, which lists only what is not picked yet', async () => {
    const user = userEvent.setup();
    const onChange = setup(['劇情']);
    await user.click(screen.getByRole('button', { name: '類型' }));
    const menu = await screen.findByRole('menu');
    expect(
      within(menu)
        .getAllByRole('menuitem')
        .map((i) => i.textContent)
    ).toEqual(['動作', '動畫']);
    await user.click(within(menu).getByRole('menuitem', { name: '動畫' }));
    expect(onChange).toHaveBeenCalledWith(['劇情', '動畫']);
  });

  it('hides ＋ 類型 once every option is picked', () => {
    setup(['動作', '劇情', '動畫']);
    expect(screen.queryByRole('button', { name: '類型' })).toBeNull();
  });
});
