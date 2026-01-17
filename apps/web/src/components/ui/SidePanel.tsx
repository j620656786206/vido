import { useEffect, useCallback, useRef } from 'react';
import { X } from 'lucide-react';
import { cn } from '../../lib/utils';

export interface SidePanelProps {
  isOpen: boolean;
  onClose: () => void;
  children: React.ReactNode;
  title?: string;
  className?: string;
}

/**
 * SidePanel Component - Spotify-style slide-in panel from the right
 * AC #4: Desktop side panel (400-500px width)
 */
export function SidePanel({
  isOpen,
  onClose,
  children,
  title,
  className,
}: SidePanelProps) {
  const panelRef = useRef<HTMLDivElement>(null);

  // Task 3.5: Close on Escape key
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
      // Prevent body scroll when panel is open
      document.body.style.overflow = 'hidden';
    }

    return () => {
      document.removeEventListener('keydown', handleKeyDown);
      document.body.style.overflow = '';
    };
  }, [isOpen, handleKeyDown]);

  // Task 3.4: Click outside to close
  const handleBackdropClick = (e: React.MouseEvent) => {
    if (e.target === e.currentTarget) {
      onClose();
    }
  };

  if (!isOpen) return null;

  return (
    <div
      className="fixed inset-0 z-40"
      onClick={handleBackdropClick}
      role="dialog"
      aria-modal="true"
      aria-labelledby={title ? 'side-panel-title' : undefined}
    >
      {/* Task 3.6: Backdrop overlay with blur */}
      <div
        className={cn(
          'absolute inset-0 bg-black/50 backdrop-blur-sm',
          'transition-opacity duration-200',
          isOpen ? 'opacity-100' : 'opacity-0'
        )}
        data-testid="side-panel-backdrop"
      />

      {/* Task 3.2 & 3.3: Panel with slide-in animation, 450px width on desktop */}
      <div
        ref={panelRef}
        className={cn(
          // Base styles
          'fixed right-0 top-0 z-50 h-full',
          // Task 3.3: Width - full on mobile, 450px on desktop
          'w-full sm:w-[450px]',
          // Background and shadow
          'bg-slate-900 shadow-2xl',
          // Task 3.2: Slide-in animation from right (300ms)
          'transform transition-transform duration-300 ease-out',
          isOpen ? 'translate-x-0' : 'translate-x-full',
          className
        )}
        data-testid="side-panel"
      >
        {/* Header with close button */}
        <div className="flex items-center justify-between border-b border-slate-800 px-4 py-3">
          {title && (
            <h2
              id="side-panel-title"
              className="text-lg font-semibold text-white"
            >
              {title}
            </h2>
          )}
          {/* Task 3.4: Close button */}
          <button
            onClick={onClose}
            className={cn(
              'rounded-lg p-2 text-gray-400',
              'hover:bg-slate-800 hover:text-white',
              'focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500',
              'transition-colors',
              // Position close button on the right if no title
              !title && 'ml-auto'
            )}
            aria-label="關閉"
            data-testid="side-panel-close"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Scrollable content area */}
        <div className="h-[calc(100%-56px)] overflow-y-auto">
          {children}
        </div>
      </div>
    </div>
  );
}

export default SidePanel;
