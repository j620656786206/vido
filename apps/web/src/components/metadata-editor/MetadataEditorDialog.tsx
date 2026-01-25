/**
 * MetadataEditorDialog Component (Story 3.8 - AC1, AC4)
 * Dialog for manually editing metadata of movies and series
 */

import { useEffect, useCallback } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { X, Loader2, Save } from 'lucide-react';
import { cn } from '../../lib/utils';
import { useUpdateMetadata } from '../../hooks/useMetadataEditor';

// Validation schema following story requirements (AC4)
const metadataSchema = z.object({
  title: z.string().min(1, '標題為必填'),
  titleEnglish: z.string().optional(),
  year: z.number().min(1900, '年份必須大於 1900').max(2100, '年份必須小於 2100'),
  genres: z.array(z.string()),
  director: z.string().optional(),
  cast: z.array(z.string()),
  overview: z.string().optional(),
  posterUrl: z.string().optional(),
});

export type MetadataFormData = z.infer<typeof metadataSchema>;

export interface MediaMetadata {
  id: string;
  mediaType: 'movie' | 'series';
  title: string;
  titleEnglish?: string;
  year?: number;
  genres?: string[];
  director?: string;
  cast?: string[];
  overview?: string;
  posterUrl?: string;
}

export interface MetadataEditorDialogProps {
  isOpen: boolean;
  onClose: () => void;
  mediaId: string;
  mediaType: 'movie' | 'series';
  initialData: MediaMetadata;
  onSuccess: () => void;
}

// Genre options following project conventions
const GENRE_OPTIONS = [
  { value: 'action', label: '動作' },
  { value: 'adventure', label: '冒險' },
  { value: 'animation', label: '動畫' },
  { value: 'comedy', label: '喜劇' },
  { value: 'crime', label: '犯罪' },
  { value: 'documentary', label: '紀錄片' },
  { value: 'drama', label: '劇情' },
  { value: 'family', label: '家庭' },
  { value: 'fantasy', label: '奇幻' },
  { value: 'history', label: '歷史' },
  { value: 'horror', label: '恐怖' },
  { value: 'music', label: '音樂' },
  { value: 'mystery', label: '懸疑' },
  { value: 'romance', label: '愛情' },
  { value: 'sci-fi', label: '科幻' },
  { value: 'thriller', label: '驚悚' },
  { value: 'war', label: '戰爭' },
  { value: 'western', label: '西部' },
];

export function MetadataEditorDialog({
  isOpen,
  onClose,
  mediaId,
  mediaType,
  initialData,
  onSuccess,
}: MetadataEditorDialogProps) {
  const updateMutation = useUpdateMetadata();

  const {
    register,
    handleSubmit,
    formState: { errors, isDirty },
    reset,
    watch,
    setValue,
  } = useForm<MetadataFormData>({
    resolver: zodResolver(metadataSchema),
    defaultValues: {
      title: initialData.title || '',
      titleEnglish: initialData.titleEnglish || '',
      year: initialData.year || new Date().getFullYear(),
      genres: initialData.genres || [],
      director: initialData.director || '',
      cast: initialData.cast || [],
      overview: initialData.overview || '',
      posterUrl: initialData.posterUrl || '',
    },
  });

  // Reset form when dialog opens with new data
  useEffect(() => {
    if (isOpen) {
      reset({
        title: initialData.title || '',
        titleEnglish: initialData.titleEnglish || '',
        year: initialData.year || new Date().getFullYear(),
        genres: initialData.genres || [],
        director: initialData.director || '',
        cast: initialData.cast || [],
        overview: initialData.overview || '',
        posterUrl: initialData.posterUrl || '',
      });
    }
  }, [isOpen, initialData, reset]);

  const selectedGenres = watch('genres');
  const castList = watch('cast');

  const onSubmit = async (data: MetadataFormData) => {
    try {
      await updateMutation.mutateAsync({
        id: mediaId,
        mediaType,
        title: data.title,
        titleEnglish: data.titleEnglish,
        year: data.year,
        genres: data.genres,
        director: data.director,
        cast: data.cast,
        overview: data.overview,
        posterUrl: data.posterUrl,
      });

      onSuccess();
      onClose();
    } catch (err) {
      console.error('Failed to update metadata:', err);
    }
  };

  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose();
      }
    },
    [onClose]
  );

  useEffect(() => {
    if (isOpen) {
      document.addEventListener('keydown', handleKeyDown);
      document.body.style.overflow = 'hidden';
    }

    return () => {
      document.removeEventListener('keydown', handleKeyDown);
      document.body.style.overflow = '';
    };
  }, [isOpen, handleKeyDown]);

  const handleBackdropClick = (e: React.MouseEvent) => {
    if (e.target === e.currentTarget) {
      onClose();
    }
  };

  const toggleGenre = (genre: string) => {
    const current = selectedGenres || [];
    if (current.includes(genre)) {
      setValue(
        'genres',
        current.filter((g) => g !== genre),
        { shouldDirty: true }
      );
    } else {
      setValue('genres', [...current, genre], { shouldDirty: true });
    }
  };

  const addCastMember = (name: string) => {
    if (name.trim() && !castList?.includes(name.trim())) {
      setValue('cast', [...(castList || []), name.trim()], { shouldDirty: true });
    }
  };

  const removeCastMember = (name: string) => {
    setValue(
      'cast',
      (castList || []).filter((c) => c !== name),
      { shouldDirty: true }
    );
  };

  if (!isOpen) return null;

  return (
    <div
      className="fixed inset-0 z-50"
      onClick={handleBackdropClick}
      role="dialog"
      aria-modal="true"
      aria-labelledby="metadata-editor-title"
    >
      {/* Backdrop */}
      <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" />

      {/* Dialog */}
      <div
        className={cn(
          'fixed left-1/2 top-1/2 z-50 -translate-x-1/2 -translate-y-1/2',
          'w-[90vw] max-w-2xl max-h-[85vh]',
          'bg-slate-900 rounded-xl shadow-2xl',
          'flex flex-col overflow-hidden'
        )}
        data-testid="metadata-editor-dialog"
      >
        {/* Header */}
        <div className="flex items-center justify-between border-b border-slate-700 px-6 py-4">
          <h2 id="metadata-editor-title" className="text-xl font-semibold text-white">
            編輯媒體資訊
          </h2>
          <button
            onClick={onClose}
            className={cn(
              'rounded-lg p-2 text-gray-400',
              'hover:bg-slate-800 hover:text-white',
              'focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500',
              'transition-colors'
            )}
            aria-label="關閉"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Form */}
        <form
          onSubmit={handleSubmit(onSubmit)}
          className="flex-1 overflow-y-auto px-6 py-4 space-y-4"
        >
          {/* Title (Chinese) */}
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1">
              標題（中文）<span className="text-red-400">*</span>
            </label>
            <input
              type="text"
              {...register('title')}
              className={cn(
                'w-full px-4 py-2',
                'bg-slate-800 border rounded-lg',
                'text-white placeholder-slate-400',
                'focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent',
                'transition-colors',
                errors.title ? 'border-red-500' : 'border-slate-700'
              )}
              placeholder="輸入中文標題"
            />
            {errors.title && <p className="mt-1 text-sm text-red-400">{errors.title.message}</p>}
          </div>

          {/* Title (English) */}
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1">標題（英文）</label>
            <input
              type="text"
              {...register('titleEnglish')}
              className={cn(
                'w-full px-4 py-2',
                'bg-slate-800 border border-slate-700 rounded-lg',
                'text-white placeholder-slate-400',
                'focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent',
                'transition-colors'
              )}
              placeholder="輸入英文標題"
            />
          </div>

          {/* Year */}
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1">
              年份 <span className="text-red-400">*</span>
            </label>
            <input
              type="number"
              {...register('year', { valueAsNumber: true })}
              className={cn(
                'w-full px-4 py-2',
                'bg-slate-800 border rounded-lg',
                'text-white placeholder-slate-400',
                'focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent',
                'transition-colors',
                errors.year ? 'border-red-500' : 'border-slate-700'
              )}
              placeholder="輸入年份"
              min={1900}
              max={2100}
            />
            {errors.year && <p className="mt-1 text-sm text-red-400">{errors.year.message}</p>}
          </div>

          {/* Genres */}
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-2">類型</label>
            <div className="flex flex-wrap gap-2">
              {GENRE_OPTIONS.map((genre) => (
                <button
                  key={genre.value}
                  type="button"
                  onClick={() => toggleGenre(genre.value)}
                  className={cn(
                    'px-3 py-1.5 rounded-full text-sm font-medium transition-colors',
                    selectedGenres?.includes(genre.value)
                      ? 'bg-blue-600 text-white'
                      : 'bg-slate-800 text-slate-400 hover:bg-slate-700 hover:text-white'
                  )}
                >
                  {genre.label}
                </button>
              ))}
            </div>
          </div>

          {/* Director */}
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1">導演</label>
            <input
              type="text"
              {...register('director')}
              className={cn(
                'w-full px-4 py-2',
                'bg-slate-800 border border-slate-700 rounded-lg',
                'text-white placeholder-slate-400',
                'focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent',
                'transition-colors'
              )}
              placeholder="輸入導演名稱"
            />
          </div>

          {/* Cast */}
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1">演員</label>
            <div className="flex flex-wrap gap-2 mb-2">
              {castList?.map((actor) => (
                <span
                  key={actor}
                  className="inline-flex items-center gap-1 px-2 py-1 bg-slate-800 text-white rounded-lg text-sm"
                >
                  {actor}
                  <button
                    type="button"
                    onClick={() => removeCastMember(actor)}
                    className="text-slate-400 hover:text-red-400 transition-colors"
                  >
                    <X className="h-3 w-3" />
                  </button>
                </span>
              ))}
            </div>
            <input
              type="text"
              placeholder="輸入演員名稱後按 Enter"
              className={cn(
                'w-full px-4 py-2',
                'bg-slate-800 border border-slate-700 rounded-lg',
                'text-white placeholder-slate-400',
                'focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent',
                'transition-colors'
              )}
              onKeyDown={(e) => {
                if (e.key === 'Enter') {
                  e.preventDefault();
                  const input = e.target as HTMLInputElement;
                  addCastMember(input.value);
                  input.value = '';
                }
              }}
            />
          </div>

          {/* Overview */}
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1">簡介</label>
            <textarea
              {...register('overview')}
              rows={4}
              className={cn(
                'w-full px-4 py-2',
                'bg-slate-800 border border-slate-700 rounded-lg',
                'text-white placeholder-slate-400',
                'focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent',
                'transition-colors resize-none'
              )}
              placeholder="輸入媒體簡介"
            />
          </div>

          {/* Poster URL */}
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1">海報圖片網址</label>
            <input
              type="text"
              {...register('posterUrl')}
              className={cn(
                'w-full px-4 py-2',
                'bg-slate-800 border border-slate-700 rounded-lg',
                'text-white placeholder-slate-400',
                'focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent',
                'transition-colors'
              )}
              placeholder="輸入海報圖片網址"
            />
          </div>
        </form>

        {/* Footer */}
        <div className="flex items-center justify-between border-t border-slate-700 px-6 py-4">
          {updateMutation.error && (
            <p className="text-sm text-red-400">更新失敗：{updateMutation.error.message}</p>
          )}
          <div className="flex gap-3 ml-auto">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 rounded-lg text-slate-300 hover:bg-slate-800 transition-colors"
            >
              取消
            </button>
            <button
              type="submit"
              onClick={handleSubmit(onSubmit)}
              disabled={updateMutation.isPending || !isDirty}
              className={cn(
                'px-4 py-2 rounded-lg bg-blue-600 text-white',
                'hover:bg-blue-700 transition-colors',
                'disabled:opacity-50 disabled:cursor-not-allowed',
                'flex items-center gap-2'
              )}
            >
              {updateMutation.isPending ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <Save className="h-4 w-4" />
              )}
              儲存
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

export default MetadataEditorDialog;
