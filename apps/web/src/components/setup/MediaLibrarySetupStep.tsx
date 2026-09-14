// Design ref: ux-design.pen Screen N3-D (TyjL0) · N3-M (YyaqL)
// 每一列＝一個媒體庫、一條路徑。完成時後端拿資料夾名稱當媒體庫名稱建立
// （setup_service.go 的 CompleteSetup），所以這跟設定頁「具名媒體庫＋多路徑」是
// 同一個資料模型，只是精靈刻意不問名字、一庫只收一條路徑。要改名或替同一個媒體庫
// 加第二條路徑，去「設定 › 媒體庫掃描」。（dsr-13 查證後撤銷原本的「不一致」警告）
import { useEffect, useId } from 'react';
import { Plus, Trash2 } from 'lucide-react';
import type { StepProps } from './SetupWizard';
import { StepNav } from './StepNav';
import { newLocalId } from '../../utils/uid';

const CONTENT_TYPES = [
  { value: 'movie', label: '電影' },
  { value: 'series', label: '影集' },
] as const;

type ContentType = (typeof CONTENT_TYPES)[number]['value'];

interface LibraryEntry {
  id: string;
  path: string;
  contentType: ContentType;
}

export function MediaLibrarySetupStep({ data, onUpdate, onNext, onBack }: StepProps) {
  // Radio group names must be unique per MOUNT, not just per row: two mounted
  // steps with the same row ids (the gallery renders one per visual state) would
  // otherwise merge into one browser radio group and steal each other's tab stop.
  const groupPrefix = useId();
  const defaultLibrary: LibraryEntry = {
    // newLocalId, NOT crypto.randomUUID: NAS installs run over http://<LAN-IP>
    // (insecure origin) where randomUUID does not exist — the first Synology
    // install crashed the wizard right here.
    id: newLocalId(),
    path: '',
    contentType: 'movie',
  };
  const libraries: LibraryEntry[] = (data.libraries as LibraryEntry[] | undefined) || [
    defaultLibrary,
  ];

  useEffect(() => {
    if (!data.libraries) {
      onUpdate({ libraries: [defaultLibrary] } as Record<string, unknown>);
    }
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  const updateLibraries = (updated: LibraryEntry[]) => {
    onUpdate({ libraries: updated } as Record<string, unknown>);
  };

  const addLibrary = () => {
    updateLibraries([...libraries, { id: newLocalId(), path: '', contentType: 'movie' }]);
  };

  const removeLibrary = (index: number) => {
    if (libraries.length <= 1) return;
    updateLibraries(libraries.filter((_, i) => i !== index));
  };

  const updateEntry = (index: number, patch: Partial<Omit<LibraryEntry, 'id'>>) => {
    updateLibraries(libraries.map((lib, i) => (i === index ? { ...lib, ...patch } : lib)));
  };

  const hasEmptyPath = libraries.some((lib) => !lib.path.trim());

  return (
    <div className="flex flex-col gap-4" data-testid="media-library-step">
      <h2 className="text-lg font-semibold text-[var(--text-primary)]">媒體庫設定</h2>
      <p className="text-sm text-[var(--text-secondary)]">
        設定您的媒體資料夾路徑和類型。至少需要一個媒體庫。
      </p>

      {libraries.map((lib, index) => (
        // The row IS the field: the design draws no inner input box, so the
        // row's own hairline turns to the focus colour while its path is being
        // typed (the type toggles carry their own ring).
        <div
          key={lib.id}
          className="flex flex-col gap-2 rounded-[var(--radius-md)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] p-3 has-[input[type=text]:focus]:border-[var(--focus-ring)]"
          data-testid={`library-entry-${index}`}
        >
          <div className="flex items-center gap-2">
            <label htmlFor={`library-path-input-${index}`} className="sr-only">
              資料夾路徑
            </label>
            <input
              id={`library-path-input-${index}`}
              type="text"
              value={lib.path}
              onChange={(e) => updateEntry(index, { path: e.target.value })}
              placeholder="/media/movies"
              className="h-11 min-w-0 flex-1 bg-transparent font-mono text-sm text-[var(--text-primary)] placeholder:text-[var(--text-muted)] focus:outline-none sm:h-8"
              data-testid={`library-path-${index}`}
            />
            {libraries.length > 1 && (
              // Neutral at rest: removing an unsaved row is not「壞了」, so it
              // does not wear 硃砂 (DESIGN.md 固定詞彙). Ghost icon button.
              <button
                type="button"
                onClick={() => removeLibrary(index)}
                className="flex h-11 w-11 shrink-0 items-center justify-center rounded-[var(--radius-md)] text-[var(--text-muted)] transition-colors hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-primary)] active:bg-[var(--bg-tertiary)] sm:h-8 sm:w-8"
                data-testid={`library-remove-${index}`}
                aria-label="移除此媒體庫"
              >
                <Trash2 className="h-4 w-4" aria-hidden="true" />
              </button>
            )}
          </div>

          <div
            role="radiogroup"
            aria-label="類型"
            className="flex gap-2"
            data-testid={`library-type-${index}`}
          >
            {CONTENT_TYPES.map((option) => {
              const checked = lib.contentType === option.value;
              return (
                <label
                  key={option.value}
                  className={`flex h-11 flex-1 cursor-pointer items-center justify-center rounded-[var(--radius-sm)] border px-3 text-xs font-semibold transition-colors has-[:focus-visible]:ring-2 has-[:focus-visible]:ring-[var(--focus-ring)] sm:h-8 sm:flex-none ${
                    checked
                      ? 'border-[var(--accent-primary)] bg-[var(--accent-subtle)] text-[var(--accent-text)]'
                      : 'border-transparent bg-[var(--bg-tertiary)] text-[var(--text-secondary)] hover:text-[var(--text-primary)]'
                  }`}
                >
                  <input
                    type="radio"
                    name={`${groupPrefix}-type-${lib.id}`}
                    value={option.value}
                    checked={checked}
                    onChange={() => updateEntry(index, { contentType: option.value })}
                    className="sr-only"
                    data-testid={`library-type-${index}-${option.value}`}
                  />
                  {option.label}
                </label>
              );
            })}
          </div>
        </div>
      ))}

      <button
        type="button"
        onClick={addLibrary}
        className="flex h-11 w-full items-center justify-center gap-2 rounded-[var(--radius-md)] border border-[var(--border-subtle)] text-sm font-semibold text-[var(--text-secondary)] transition-colors hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-primary)] active:bg-[var(--bg-tertiary)]"
        data-testid="add-library-button"
      >
        <Plus className="h-4 w-4" aria-hidden="true" />
        新增媒體庫
      </button>

      <StepNav onNext={onNext} onBack={onBack} nextDisabled={hasEmptyPath} />
    </div>
  );
}
