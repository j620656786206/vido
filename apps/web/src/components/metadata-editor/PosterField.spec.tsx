/**
 * PosterField — 修改資訊 海報 column (poster-upload-b AC #3, B′14 states)
 */
import { useState } from 'react';
import { describe, it, expect, vi, beforeAll } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {
  PosterField,
  POSTER_COPY,
  type PosterChoice,
  type PosterFieldProps,
  type PosterPhase,
} from './PosterField';

beforeAll(() => {
  URL.createObjectURL = vi.fn(() => 'blob:preview');
  URL.revokeObjectURL = vi.fn();
});

const jpeg = (bytes = 10) => new File([new Uint8Array(bytes)], 'p.jpg', { type: 'image/jpeg' });

function Harness(
  props: Partial<PosterFieldProps> & { onChoice?: (c: PosterChoice | null) => void }
) {
  const [choice, setChoice] = useState<PosterChoice | null>(props.choice ?? null);
  return (
    <PosterField
      currentPoster={'currentPoster' in props ? props.currentPoster : '/abc.jpg'}
      phase={(props.phase ?? 'idle') as PosterPhase}
      initialState={props.initialState}
      choice={choice}
      onChoiceChange={(c) => {
        setChoice(c);
        props.onChoice?.(c);
      }}
    />
  );
}

const fileInput = () => screen.getByTestId('poster-file-input') as HTMLInputElement;

describe('PosterField', () => {
  it('① shows the current poster and 換一張圖片', () => {
    render(<Harness />);
    expect(screen.getByRole('img', { name: '目前的海報' })).toHaveAttribute(
      'src',
      'https://image.tmdb.org/t/p/w342/abc.jpg'
    );
    expect(screen.getByRole('button', { name: '換一張圖片' })).toBeInTheDocument();
    expect(screen.queryByText(POSTER_COPY.pending)).toBeNull();
  });

  it('② with no poster says 還沒有海報 and offers 上傳圖片', () => {
    render(<Harness currentPoster={null} />);
    expect(screen.getByText('還沒有海報')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '上傳圖片' })).toBeInTheDocument();
  });

  it('the button opens the file picker (the keyboard path — drop is only a shortcut)', async () => {
    render(<Harness />);
    const click = vi.spyOn(fileInput(), 'click');
    await userEvent.click(screen.getByRole('button', { name: '換一張圖片' }));
    expect(click).toHaveBeenCalled();
  });

  it('④ a picked image previews with 新海報・尚未儲存 — and is only reported, not uploaded', async () => {
    const onChoice = vi.fn();
    render(<Harness onChoice={onChoice} />);
    const file = jpeg();
    await userEvent.upload(fileInput(), file);
    expect(onChoice).toHaveBeenCalledWith({ kind: 'file', file });
    expect(screen.getByRole('img', { name: '新的海報' })).toHaveAttribute('src', 'blob:preview');
    expect(screen.getByText(POSTER_COPY.pending)).toBeInTheDocument();
  });

  it('⑥ a GIF is refused on the spot and the current poster stays', () => {
    const onChoice = vi.fn();
    render(<Harness onChoice={onChoice} />);
    fireEvent.change(fileInput(), {
      target: { files: [new File(['x'], 'a.gif', { type: 'image/gif' })] },
    });
    expect(onChoice).not.toHaveBeenCalled();
    expect(screen.getByText(POSTER_COPY.badType)).toBeInTheDocument();
    expect(screen.getByRole('img', { name: '目前的海報' })).toBeInTheDocument();
  });

  it('⑦ a file over 5 MB is refused on the spot, and the message is tied to the button', () => {
    const onChoice = vi.fn();
    render(<Harness onChoice={onChoice} />);
    fireEvent.change(fileInput(), { target: { files: [jpeg(5 * 1024 * 1024 + 1)] } });
    expect(onChoice).not.toHaveBeenCalled();
    const msg = screen.getByText(POSTER_COPY.tooBig);
    expect(screen.getByRole('button', { name: '換一張圖片' })).toHaveAttribute(
      'aria-describedby',
      msg.id
    );
    expect(msg).toHaveAttribute('aria-live', 'polite');
  });

  it('③ dragging over the poster shows 放開就用這張, and a drop picks the file', () => {
    const onChoice = vi.fn();
    render(<Harness onChoice={onChoice} />);
    const target = screen.getByTestId('poster-drop-target');
    fireEvent.dragEnter(target);
    fireEvent.dragEnter(target.querySelector('img')!); // crossing into a child
    fireEvent.dragLeave(target.querySelector('img')!);
    expect(screen.getByText('放開就用這張')).toBeInTheDocument(); // no flicker
    const file = jpeg();
    fireEvent.drop(target, { dataTransfer: { files: [file] } });
    expect(screen.queryByText('放開就用這張')).toBeNull();
    expect(onChoice).toHaveBeenCalledWith({ kind: 'file', file });
  });

  it('⑤ while uploading, shows 上傳中… and locks the controls', () => {
    render(<Harness choice={{ kind: 'file', file: jpeg() }} phase="uploading" />);
    expect(screen.getByText('上傳中…')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '換一張圖片' })).toBeDisabled();
    expect(screen.getByRole('button', { name: '改用圖片網址' })).toBeDisabled();
  });

  it('⑧ a failed upload keeps the new preview and says nothing was saved', () => {
    render(<Harness choice={{ kind: 'file', file: jpeg() }} phase="failed" />);
    expect(screen.getByRole('img', { name: '新的海報' })).toBeInTheDocument();
    expect(screen.getByText(POSTER_COPY.uploadFailed)).toBeInTheDocument();
  });

  it('⑨ 改用圖片網址 swaps in a URL box; the preview follows; 改用上傳 goes back', async () => {
    const user = userEvent.setup();
    const onChoice = vi.fn();
    render(<Harness onChoice={onChoice} />);
    await user.click(screen.getByRole('button', { name: '改用圖片網址' }));
    await user.type(screen.getByLabelText('圖片網址'), 'https://x.test/p.jpg');
    expect(onChoice).toHaveBeenLastCalledWith({
      kind: 'url',
      url: 'https://x.test/p.jpg',
      status: 'loading',
    });
    const img = screen.getByRole('img', { name: '新的海報' });
    expect(img).toHaveAttribute('src', 'https://x.test/p.jpg');

    fireEvent.error(img);
    expect(onChoice).toHaveBeenLastCalledWith({
      kind: 'url',
      url: 'https://x.test/p.jpg',
      status: 'broken',
    });
    expect(screen.getByText(POSTER_COPY.badUrl)).toBeInTheDocument();
    expect(screen.getByLabelText('圖片網址')).toHaveAttribute('aria-invalid', 'true');

    await user.click(screen.getByRole('button', { name: '改用上傳' }));
    expect(onChoice).toHaveBeenLastCalledWith(null);
    expect(screen.getByRole('button', { name: '換一張圖片' })).toBeInTheDocument();
  });

  it('a URL that loads is reported ok', async () => {
    const onChoice = vi.fn();
    render(<Harness onChoice={onChoice} />);
    await userEvent.click(screen.getByRole('button', { name: '改用圖片網址' }));
    await userEvent.type(screen.getByLabelText('圖片網址'), 'https://x.test/p.jpg');
    fireEvent.load(screen.getByRole('img', { name: '新的海報' }));
    expect(onChoice).toHaveBeenLastCalledWith({
      kind: 'url',
      url: 'https://x.test/p.jpg',
      status: 'ok',
    });
  });

  it('releases the blob preview when the file is replaced', async () => {
    render(<Harness />);
    await userEvent.upload(fileInput(), jpeg());
    await userEvent.upload(fileInput(), jpeg(20));
    expect(URL.revokeObjectURL).toHaveBeenCalledWith('blob:preview');
  });
});
