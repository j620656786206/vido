import { Link } from '@tanstack/react-router';

export function EmptyLibrary() {
  return (
    <div className="flex flex-col items-center justify-center py-24 text-center">
      <div className="mb-4 text-6xl">
        <span role="img" aria-label="媒體庫">
          📚🎬
        </span>
      </div>
      <h2 className="mb-2 text-xl font-semibold text-white">歡迎來到你的媒體庫</h2>
      <p className="mb-6 max-w-md text-slate-400">
        你的媒體庫目前是空的。開始搜尋並添加影片到你的收藏吧！
      </p>
      <Link
        to="/search"
        className="rounded-lg bg-blue-600 px-6 py-3 font-medium text-white transition-colors hover:bg-blue-700"
      >
        搜尋媒體
      </Link>
    </div>
  );
}
