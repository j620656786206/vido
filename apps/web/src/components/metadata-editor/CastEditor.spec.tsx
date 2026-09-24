/**
 * CastEditor — 修改資訊 演員 chips (poster-upload-a AC #4)
 */
import { describe, it, expect, vi } from 'vitest';
import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { CastEditor } from './CastEditor';

function setup(cast: string[]) {
  const onChange = vi.fn();
  render(
    <>
      <span id="lbl">演員</span>
      <CastEditor labelId="lbl" cast={cast} onChange={onChange} />
    </>
  );
  return onChange;
}

describe('CastEditor', () => {
  it('names the chip group by the field label', () => {
    setup(['花江夏樹']);
    expect(
      within(screen.getByRole('group', { name: '演員' })).getByText('花江夏樹')
    ).toBeInTheDocument();
  });

  it('removes an actor with its ×', async () => {
    const onChange = setup(['花江夏樹', '鬼頭明里']);
    await userEvent.click(screen.getByRole('button', { name: '移除演員：花江夏樹' }));
    expect(onChange).toHaveBeenCalledWith(['鬼頭明里']);
  });

  it('＋ 演員 opens a box; Enter adds a trimmed new name', async () => {
    const user = userEvent.setup();
    const onChange = setup(['花江夏樹']);
    await user.click(screen.getByRole('button', { name: '演員' }));
    const box = screen.getByRole('textbox', { name: '新增演員' });
    expect(box).toHaveFocus();
    await user.type(box, '  下野紘 {Enter}');
    expect(onChange).toHaveBeenCalledWith(['花江夏樹', '下野紘']);
  });

  it('ignores blanks and repeats', async () => {
    const user = userEvent.setup();
    const onChange = setup(['花江夏樹']);
    await user.click(screen.getByRole('button', { name: '演員' }));
    await user.type(screen.getByRole('textbox', { name: '新增演員' }), '   {Enter}花江夏樹{Enter}');
    expect(onChange).not.toHaveBeenCalled();
  });

  it('Esc closes the box without adding', async () => {
    const user = userEvent.setup();
    const onChange = setup([]);
    await user.click(screen.getByRole('button', { name: '演員' }));
    await user.type(screen.getByRole('textbox', { name: '新增演員' }), '下野紘{Escape}');
    expect(screen.queryByRole('textbox', { name: '新增演員' })).toBeNull();
    expect(onChange).not.toHaveBeenCalled();
  });

  it('leaving the box keeps what was typed (a click elsewhere is not a cancel)', async () => {
    const user = userEvent.setup();
    const onChange = setup([]);
    await user.click(screen.getByRole('button', { name: '演員' }));
    await user.type(screen.getByRole('textbox', { name: '新增演員' }), '下野紘');
    await user.tab();
    expect(onChange).toHaveBeenCalledWith(['下野紘']);
  });

  it('Esc really cancels — nothing is added when the box then loses focus', async () => {
    const user = userEvent.setup();
    const onChange = setup([]);
    await user.click(screen.getByRole('button', { name: '演員' }));
    await user.type(screen.getByRole('textbox', { name: '新增演員' }), '下野紘{Escape}');
    await user.tab();
    expect(onChange).not.toHaveBeenCalled();
  });
});
