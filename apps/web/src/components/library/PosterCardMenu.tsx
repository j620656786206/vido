import { useState, useRef, useEffect, useCallback } from 'react';
import { Eye, RefreshCw, Download, Trash2 } from 'lucide-react';
import { cn } from '../../lib/utils';

interface PosterCardMenuProps {
  onViewDetails: () => void;
  onReparse: () => void;
  onExport: () => void;
  onDelete: () => void;
  isOpen: boolean;
  onClose: () => void;
  isMobile?: boolean;
}

interface MenuItem {
  label: string;
  icon: React.ReactNode;
  onClick: () => void;
  variant?: 'default' | 'danger';
  separator?: boolean;
}

export function PosterCardMenu({
  onViewDetails,
  onReparse,
  onExport,
  onDelete,
  isOpen,
  onClose,
  isMobile = false,
}: PosterCardMenuProps) {
  const menuRef = useRef<HTMLDivElement>(null);
  const [showConfirm, setShowConfirm] = useState(false);

  const handleClickOutside = useCallback(
    (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        onClose();
        setShowConfirm(false);
      }
    },
    [onClose]
  );

  useEffect(() => {
    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside);
      return () => document.removeEventListener('mousedown', handleClickOutside);
    }
  }, [isOpen, handleClickOutside]);

  if (!isOpen) return null;

  const handleDelete = () => {
    if (showConfirm) {
      onDelete();
      onClose();
      setShowConfirm(false);
    } else {
      setShowConfirm(true);
    }
  };

  const menuItems: MenuItem[] = [
    {
      label: '查看詳情',
      icon: <Eye className="h-4 w-4" />,
      onClick: () => {
        onViewDetails();
        onClose();
      },
    },
    {
      label: '重新解析',
      icon: <RefreshCw className="h-4 w-4" />,
      onClick: () => {
        onReparse();
        onClose();
      },
    },
    {
      label: '匯出中繼資料',
      icon: <Download className="h-4 w-4" />,
      onClick: () => {
        onExport();
        onClose();
      },
    },
    {
      label: showConfirm ? '確認刪除' : '刪除',
      icon: <Trash2 className="h-4 w-4" />,
      onClick: handleDelete,
      variant: 'danger',
      separator: true,
    },
  ];

  if (isMobile) {
    return (
      <>
        {/* Backdrop */}
        <div className="fixed inset-0 z-40 bg-black/50" onClick={onClose} />
        {/* Bottom sheet */}
        <div
          ref={menuRef}
          data-testid="poster-card-menu"
          className="fixed inset-x-0 bottom-0 z-50 rounded-t-2xl bg-slate-800 p-4 pb-8"
        >
          <div className="mb-4 flex justify-center">
            <div className="h-1 w-10 rounded-full bg-slate-600" />
          </div>
          {menuItems.map((item, index) => (
            <div key={item.label}>
              {item.separator && <div className="my-2 border-t border-slate-700" />}
              <button
                onClick={item.onClick}
                className={cn(
                  'flex w-full items-center gap-3 rounded-lg px-4 py-3 text-left transition-colors',
                  item.variant === 'danger'
                    ? 'text-red-400 hover:bg-red-500/10'
                    : 'text-slate-200 hover:bg-slate-700'
                )}
                data-testid={`menu-item-${index}`}
              >
                {item.icon}
                <span>{item.label}</span>
              </button>
            </div>
          ))}
        </div>
      </>
    );
  }

  return (
    <div
      ref={menuRef}
      data-testid="poster-card-menu"
      className="absolute right-0 top-8 z-30 min-w-[180px] rounded-lg bg-slate-800 py-1 shadow-xl ring-1 ring-slate-700"
    >
      {menuItems.map((item, index) => (
        <div key={item.label}>
          {item.separator && <div className="my-1 border-t border-slate-700" />}
          <button
            onClick={item.onClick}
            className={cn(
              'flex w-full items-center gap-2 px-4 py-2 text-left text-sm transition-colors',
              item.variant === 'danger'
                ? 'text-red-400 hover:bg-red-500/10'
                : 'text-slate-200 hover:bg-slate-700'
            )}
            data-testid={`menu-item-${index}`}
          >
            {item.icon}
            <span>{item.label}</span>
          </button>
        </div>
      ))}
    </div>
  );
}
